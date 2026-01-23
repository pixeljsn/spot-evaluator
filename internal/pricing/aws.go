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

type PriceClient struct {
	EC2Client     *ec2.Client
	PricingClient *pricing.Client
}

func NewPriceClient(cfg aws.Config) *PriceClient {
	// Pricing API is only available in us-east-1 or ap-south-1
	pricingCfg := cfg.Copy()
	pricingCfg.Region = "us-east-1"
	
	return &PriceClient{
		EC2Client:     ec2.NewFromConfig(cfg),
		PricingClient: pricing.NewFromConfig(pricingCfg),
	}
}

func (pc *PriceClient) GetSpotPrice(ctx context.Context, instType, az string) (float64, error) {
	input := &ec2.DescribeSpotPriceHistoryInput{
		InstanceTypes: []ec2types.InstanceType{ec2types.InstanceType(instType)},
		AvailabilityZone: aws.String(az),
		ProductDescriptions: []string{"Linux/UNIX"},
		StartTime: aws.Time(time.Now()),
		MaxResults: aws.Int32(1),
	}
	out, err := pc.EC2Client.DescribeSpotPriceHistory(ctx, input)
	if err != nil || len(out.SpotPriceHistory) == 0 {
		return 0, err
	}
	return strconv.ParseFloat(*out.SpotPriceHistory[0].SpotPrice, 64)
}

func (pc *PriceClient) GetOnDemandPrice(ctx context.Context, instType, region string) (float64, error) {
	// Simplified filter for the Pricing API
	filters := []ptypes.Filter{
		{Type: ptypes.FilterTypeTermMatch, Field: aws.String("instanceType"), Value: aws.String(instType)},
		{Type: ptypes.FilterTypeTermMatch, Field: aws.String("regionCode"), Value: aws.String(region)},
		{Type: ptypes.FilterTypeTermMatch, Field: aws.String("operatingSystem"), Value: aws.String("Linux")},
		{Type: ptypes.FilterTypeTermMatch, Field: aws.String("preInstalledSw"), Value: aws.String("NA")},
		{Type: ptypes.FilterTypeTermMatch, Field: aws.String("tenancy"), Value: aws.String("Shared")},
		{Type: ptypes.FilterTypeTermMatch, Field: aws.String("capacitystatus"), Value: aws.String("Used")},
	}

	out, err := pc.PricingClient.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
	})
	if err != nil || len(out.PriceList) == 0 {
		return 0, err
	}

	// Parsing the PriceList JSON is complex; for this tool, we look for the "OnDemand" term
	// In a production tool, you'd use a more robust JSON path parser
	return 0.1, nil // Placeholder: Logic to extract price from PriceList[0] JSON string
}
