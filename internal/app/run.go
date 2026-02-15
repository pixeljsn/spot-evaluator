package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"spot-evaluator/internal/collector"
	"spot-evaluator/internal/pricing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Run executes the spot evaluator CLI workflow.
func Run() {
	clientset, err := newKubernetesClient()
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %v", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}
	priceClient := pricing.NewPriceClient(awsCfg)

	inventory, err := collector.GetInventory(clientset)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n%-15s %-12s %-8s %-12s %-12s %-12s\n", "INSTANCE", "AZ", "COUNT", "ON-DEMAND", "SPOT", "SAVINGS")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, item := range inventory {
		spot, err := priceClient.GetSpotPrice(context.Background(), item.InstanceType, item.AZ)
		if err != nil {
			log.Printf("warning: failed to get spot price for %s/%s: %v", item.InstanceType, item.AZ, err)
			continue
		}

		onDemand, err := priceClient.GetOnDemandPrice(context.Background(), item.InstanceType, item.Region)
		if err != nil {
			log.Printf("warning: failed to get on-demand price for %s/%s: %v", item.InstanceType, item.Region, err)
			continue
		}

		savings := 0.0
		if onDemand > 0 {
			savings = ((onDemand - spot) / onDemand) * 100
		}

		fmt.Printf("%-15s %-12s %-8d $%-11.4f $%-11.4f %-6.2f%%\n", item.InstanceType, item.AZ, item.Count, onDemand, spot, savings)

		alternatives, err := priceClient.GetReplacementOptions(context.Background(), item, 3)
		if err != nil {
			log.Printf("warning: failed to find alternatives for %s/%s: %v", item.InstanceType, item.AZ, err)
			continue
		}

		if len(alternatives) == 0 {
			fmt.Println("  no lower-cost compatible replacements found")
			continue
		}

		fmt.Println("  replacements with additional spot savings ($/hour):")
		for _, alt := range alternatives {
			fmt.Printf("  - %-12s spot=$%-10.4f save/node=$%-8.4f save/group=$%-8.4f\n",
				alt.InstanceType, alt.SpotPrice, alt.SavingsPerNodePerHour, alt.SavingsPerGroupPerHour)
		}
	}
}

func newKubernetesClient() (*kubernetes.Clientset, error) {
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
