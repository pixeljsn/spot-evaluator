package collector

import (
    "context"
    "spot-evaluator/pkg/models"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

func GetInventory(clientset *kubernetes.Clientset) ([]models.NodeGroup, error) {
    nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return nil, err
    }

    inventoryMap := make(map[string]*models.NodeGroup)

    for _, node := range nodes.Items {
        instanceType := node.Labels["node.kubernetes.io/instance-type"]
        az := node.Labels["topology.kubernetes.io/zone"]
        region := node.Labels["topology.kubernetes.io/region"]
        
        // AWS EKS often uses this label for capacity type
        isSpot := node.Labels["eks.amazonaws.com/capacityType"] == "SPOT"

        key := instanceType + "-" + az
        if _, exists := inventoryMap[key]; !exists {
            inventoryMap[key] = &models.NodeGroup{
                InstanceType: instanceType,
                AZ:           az,
                Region:       region,
                Count:        0,
                IsSpot:       isSpot,
            }
        }
        inventoryMap[key].Count++
    }

    // Convert map to slice
    result := make([]models.NodeGroup, 0, len(inventoryMap))
    for _, group := range inventoryMap {
        result = append(result, *group)
    }
    return result, nil
}
