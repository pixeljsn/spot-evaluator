package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"spot-evaluator/internal/collector"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 1. Determine Kubeconfig path
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// 2. Build configuration from the path
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	// 3. Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating kubernetes client: %v", err)
	}

	// 4. Run the Collector
	fmt.Println("🔍 Fetching node inventory from cluster...")
	inventory, err := collector.GetInventory(clientset)
	if err != nil {
		log.Fatalf("Error collecting inventory: %v", err)
	}

	// 5. Iterative Test: Print Results
	fmt.Printf("\n%-20s %-15s %-10s %-10s\n", "INSTANCE TYPE", "AZ", "COUNT", "CAPACITY")
	fmt.Println("------------------------------------------------------------")
	for _, item := range inventory {
		capacity := "On-Demand"
		if item.IsSpot {
			capacity = "Spot"
		}
		fmt.Printf("%-20s %-15s %-10d %-10s\n", item.InstanceType, item.AZ, item.Count, capacity)
	}
}
