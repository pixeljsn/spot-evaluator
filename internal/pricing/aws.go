package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"spot-evaluator/pkg/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	ptypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

const linuxUnixProductDescription = "Linux/UNIX"

type PriceClient struct {
	EC2Client     *ec2.Client
	PricingClient *pricing.Client
}

func NewPriceClient(cfg aws.Config) *PriceClient {
	pricingCfg := cfg.Copy()
	pricingCfg.Region = "us-east-1"

	return &PriceClient{
		EC2Client:     ec2.NewFromConfig(cfg),
		PricingClient: pricing.NewFromConfig(pricingCfg),
	}
}

func (pc *PriceClient) GetSpotPrice(ctx context.Context, instType, az string) (float64, error) {
	input := &ec2.DescribeSpotPriceHistoryInput{
		InstanceTypes:       []ec2types.InstanceType{ec2types.InstanceType(instType)},
		AvailabilityZone:    aws.String(az),
		ProductDescriptions: []string{linuxUnixProductDescription},
		StartTime:           aws.Time(time.Now()),
		MaxResults:          aws.Int32(1),
	}
	out, err := pc.EC2Client.DescribeSpotPriceHistory(ctx, input)
	if err != nil {
		return 0, err
	}
	if len(out.SpotPriceHistory) == 0 {
		return 0, fmt.Errorf("no spot history returned for %s in %s", instType, az)
	}

	return strconv.ParseFloat(aws.ToString(out.SpotPriceHistory[0].SpotPrice), 64)
}

func (pc *PriceClient) GetOnDemandPrice(ctx context.Context, instType, region string) (float64, error) {
	filters := []ptypes.Filter{
		{Field: aws.String("ServiceCode"), Type: ptypes.FilterTypeTermMatch, Value: aws.String("AmazonEC2")},
		{Field: aws.String("instanceType"), Type: ptypes.FilterTypeTermMatch, Value: aws.String(instType)},
		{Field: aws.String("regionCode"), Type: ptypes.FilterTypeTermMatch, Value: aws.String(region)},
		{Field: aws.String("operatingSystem"), Type: ptypes.FilterTypeTermMatch, Value: aws.String("Linux")},
		{Field: aws.String("tenancy"), Type: ptypes.FilterTypeTermMatch, Value: aws.String("Shared")},
		{Field: aws.String("preInstalledSw"), Type: ptypes.FilterTypeTermMatch, Value: aws.String("NA")},
		{Field: aws.String("capacitystatus"), Type: ptypes.FilterTypeTermMatch, Value: aws.String("Used")},
	}

	out, err := pc.PricingClient.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
	})
	if err != nil {
		return 0, err
	}
	if len(out.PriceList) == 0 {
		return 0, fmt.Errorf("no on-demand price found for %s", instType)
	}

	return parseOnDemandUSDPrice(out.PriceList[0])
}

func parseOnDemandUSDPrice(priceListItem string) (float64, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(priceListItem), &payload); err != nil {
		return 0, fmt.Errorf("failed to decode pricing payload: %w", err)
	}

	terms, ok := payload["terms"].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("pricing payload missing terms block")
	}

	onDemand, ok := terms["OnDemand"].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("pricing payload missing OnDemand terms")
	}

	for _, skuTermRaw := range onDemand {
		skuTerm, ok := skuTermRaw.(map[string]any)
		if !ok {
			continue
		}

		priceDimensions, ok := skuTerm["priceDimensions"].(map[string]any)
		if !ok {
			continue
		}

		for _, dimensionRaw := range priceDimensions {
			dimension, ok := dimensionRaw.(map[string]any)
			if !ok {
				continue
			}

			pricePerUnit, ok := dimension["pricePerUnit"].(map[string]any)
			if !ok {
				continue
			}

			usdRaw, ok := pricePerUnit["USD"]
			if !ok {
				continue
			}

			usd, ok := usdRaw.(string)
			if !ok || usd == "" {
				continue
			}

			price, err := strconv.ParseFloat(usd, 64)
			if err != nil {
				continue
			}
			return price, nil
		}
	}

	return 0, fmt.Errorf("could not find on-demand USD price dimension")
}

func (pc *PriceClient) GetReplacementOptions(ctx context.Context, group models.NodeGroup, limit int) ([]models.ReplacementOption, error) {
	currentSpec, err := pc.describeInstanceType(ctx, group.InstanceType)
	if err != nil {
		return nil, err
	}

	currentSpotPrice, err := pc.GetSpotPrice(ctx, group.InstanceType, group.AZ)
	if err != nil {
		return nil, err
	}

	offered, err := pc.listInstanceTypeOfferings(ctx, group.AZ)
	if err != nil {
		return nil, err
	}

	candidateTypes := make([]string, 0, len(offered))
	for _, offeredType := range offered {
		if offeredType == group.InstanceType {
			continue
		}
		candidateTypes = append(candidateTypes, offeredType)
	}

	options := make([]models.ReplacementOption, 0)
	for _, candidateType := range candidateTypes {
		spec, err := pc.describeInstanceType(ctx, candidateType)
		if err != nil || !isReplacementCompatible(currentSpec, spec) {
			continue
		}

		candidateSpot, err := pc.GetSpotPrice(ctx, candidateType, group.AZ)
		if err != nil || candidateSpot >= currentSpotPrice {
			continue
		}

		savingsPerNode := currentSpotPrice - candidateSpot
		options = append(options, models.ReplacementOption{
			InstanceType:           candidateType,
			SpotPrice:              candidateSpot,
			SavingsPerNodePerHour:  savingsPerNode,
			SavingsPerGroupPerHour: savingsPerNode * float64(group.Count),
		})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].SavingsPerGroupPerHour > options[j].SavingsPerGroupPerHour
	})

	if len(options) > limit {
		return options[:limit], nil
	}

	return options, nil
}

func (pc *PriceClient) listInstanceTypeOfferings(ctx context.Context, az string) ([]string, error) {
	paginator := ec2.NewDescribeInstanceTypeOfferingsPaginator(pc.EC2Client, &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: ec2types.LocationTypeAvailabilityZone,
		Filters: []ec2types.Filter{{
			Name:   aws.String("location"),
			Values: []string{az},
		}},
	})

	offeredMap := make(map[string]struct{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, offering := range page.InstanceTypeOfferings {
			offeredMap[string(offering.InstanceType)] = struct{}{}
		}
	}

	offered := make([]string, 0, len(offeredMap))
	for instType := range offeredMap {
		offered = append(offered, instType)
	}

	return offered, nil
}

func (pc *PriceClient) describeInstanceType(ctx context.Context, instType string) (ec2types.InstanceTypeInfo, error) {
	out, err := pc.EC2Client.DescribeInstanceTypes(ctx, &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []ec2types.InstanceType{ec2types.InstanceType(instType)},
	})
	if err != nil {
		return ec2types.InstanceTypeInfo{}, err
	}
	if len(out.InstanceTypes) == 0 {
		return ec2types.InstanceTypeInfo{}, fmt.Errorf("instance type %s not found", instType)
	}
	return out.InstanceTypes[0], nil
}

func isReplacementCompatible(current, candidate ec2types.InstanceTypeInfo) bool {
	if aws.ToBool(candidate.BareMetal) || aws.ToBool(candidate.FreeTierEligible) {
		return false
	}

	if current.VCpuInfo == nil || candidate.VCpuInfo == nil || current.MemoryInfo == nil || candidate.MemoryInfo == nil {
		return false
	}

	if aws.ToInt32(candidate.VCpuInfo.DefaultVCpus) < aws.ToInt32(current.VCpuInfo.DefaultVCpus) {
		return false
	}

	if aws.ToInt64(candidate.MemoryInfo.SizeInMiB) < aws.ToInt64(current.MemoryInfo.SizeInMiB) {
		return false
	}

	return true
}
