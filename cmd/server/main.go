package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"spot-evaluator/internal/collector"
	"spot-evaluator/internal/pricing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// K8s Setup
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	// AWS Setup
	awsCfg, _ := awsconfig.LoadDefaultConfig(context.TODO())
	priceClient := pricing.NewPriceClient(awsCfg)

	inventory, err := collector.GetInventory(clientset)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n%-15s %-12s %-8s %-15s\n", "INSTANCE", "AZ", "COUNT", "SPOT PRICE")
	fmt.Println("---------------------------------------------------------")

	for _, item := range inventory {
		spot, _ := priceClient.GetSpotPrice(context.TODO(), item.InstanceType, item.AZ)
		fmt.Printf("%-15s %-12s %-8d $%-15.4f\n", item.InstanceType, item.AZ, item.Count, spot)
	}
}
