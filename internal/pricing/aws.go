package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	ptypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// PriceListResponse handles the nested dynamic keys from AWS JSON
type PriceListResponse struct {
	Terms struct {
		OnDemand map[string]map[string]struct {
			PriceDimensions map[string]struct {
				Unit         string            `json:"unit"`
				PricePerUnit map[string]string `json:"pricePerUnit"`
			} `json:"priceDimensions"`
		} `json:"OnDemand"`
	} `json:"terms"`
}

type PriceClient struct {
	EC2Client     *ec2.Client
	PricingClient *pricing.Client
}

func NewPriceClient(cfg aws.Config) *PriceClient {
	// Pricing API ONLY works in us-east-1
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
		ProductDescriptions: []string{"Linux/UNIX"},
		StartTime:           aws.Time(time.Now()),
		MaxResults:          aws.Int32(1),
	}
	out, err := pc.EC2Client.DescribeSpotPriceHistory(ctx, input)
	if err != nil || len(out.SpotPriceHistory) == 0 {
		return 0, err
	}
	return strconv.ParseFloat(*out.SpotPriceHistory[0].SpotPrice, 64)
}

func (pc *PriceClient) GetOnDemandPrice(ctx context.Context, instType, region string) (float64, error) {
	// Map region code (us-east-1) to Location Name (US East (N. Virginia))
	// In a full tool, use a helper map. For testing, we use GetProducts filters.
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
	if err != nil || len(out.PriceList) == 0 {
		return 0, fmt.Errorf("no on-demand price found for %s", instType)
	}

	// Double Parse: out.PriceList[0] is a JSON string
	var priceData PriceListResponse
	if err := json.Unmarshal([]byte(out.PriceList[0]), &priceData); err != nil {
		return 0, err
	}

	// Walk the dynamic map: SKU -> OfferID -> DimensionID
	for _, skuMap := range priceData.Terms.OnDemand {
		for _, offer := range skuMap {
			for _, dimension := range offer.PriceDimensions {
				if val, ok := dimension.PricePerUnit["USD"]; ok {
					return strconv.ParseFloat(val, 64)
				}
			}
		}
	}

	return 0, fmt.Errorf("could not find price dimension")
}
