package main

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

const (
	CPUWarning     = 70.0
	CPUCritical    = 85.0
	MemoryWarning  = 75.0
	MemoryCritical = 90.0
)

func main() {
	ctx := context.Background()

	// Load Kubernetes configuration
	config, err := clientcmd.BuildConfigFromFlags(
		"",
		clientcmd.RecommendedHomeFile,
	)
	if err != nil {
		fmt.Printf("Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Create Metrics API client
	metricsClient, err := metricsclient.NewForConfig(config)
	if err != nil {
		fmt.Printf("Failed to create Metrics client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to Kubernetes successfully!")

	// Fetch node information
	nodes, err := clientset.CoreV1().Nodes().List(
		ctx,
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("Failed to retrieve nodes: %v\n", err)
		os.Exit(1)
	}

	// Fetch live node metrics
	nodeMetrics, err := metricsClient.MetricsV1beta1().
		NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to retrieve node metrics: %v\n", err)
		os.Exit(1)
	}

	// Map node metrics by node name
	metricsMap := make(map[string]corev1.ResourceList)

	for _, metric := range nodeMetrics.Items {
		metricsMap[metric.Name] = metric.Usage
	}

	fmt.Printf("\nTotal Nodes: %d\n", len(nodes.Items))

	// Display node information
	for _, node := range nodes.Items {
		displayNodeInformation(node, metricsMap)
	}

	// Fetch pod specifications (requests and limits)
	pods, err := clientset.CoreV1().Pods("").
		List(ctx, metav1.ListOptions{})

	if err != nil {
		fmt.Printf("Failed to retrieve pods: %v\n", err)
		os.Exit(1)
	}

	// Map pods using namespace/name
	podMap := make(map[string]corev1.Pod)

	for _, pod := range pods.Items {
		key := pod.Namespace + "/" + pod.Name
		podMap[key] = pod
	}

	// Fetch pod metrics across all namespaces
	podMetrics, err := metricsClient.MetricsV1beta1().
		PodMetricses("").
		List(ctx, metav1.ListOptions{})

	if err != nil {
		fmt.Printf("Failed to retrieve pod metrics: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nTotal Pods with Metrics: %d\n", len(podMetrics.Items))

	for _, pod := range podMetrics.Items {

		key := pod.Namespace + "/" + pod.Name
		podSpec := podMap[key]

		var cpuRequest, cpuLimit int64
		var memoryRequest, memoryLimit int64

		for _, container := range podSpec.Spec.Containers {
			requests := container.Resources.Requests
			limits := container.Resources.Limits

			if value, ok := requests[corev1.ResourceCPU]; ok {
				cpuRequest += value.MilliValue()
			}

			if value, ok := limits[corev1.ResourceCPU]; ok {
				cpuLimit += value.MilliValue()
			}

			if value, ok := requests[corev1.ResourceMemory]; ok {
				memoryRequest += value.Value()
			}

			if value, ok := limits[corev1.ResourceMemory]; ok {
				memoryLimit += value.Value()
			}
		}

		var cpuMilli int64
		var memoryBytes int64

		// Aggregate resource usage across containers
		for _, container := range pod.Containers {
			cpuMilli += container.Usage.Cpu().MilliValue()
			memoryBytes += container.Usage.Memory().Value()
		}

		fmt.Println("\n-----------------------------")
		fmt.Printf("Namespace: %s\n", pod.Namespace)
		fmt.Printf("Pod Name: %s\n", pod.Name)
		fmt.Printf("CPU Usage: %dm\n", cpuMilli)
		fmt.Printf("Memory Usage: %.2f MiB\n",
			float64(memoryBytes)/(1024*1024))
		// Calculate utilization percentages
		printUtilization := func(label string, usage, capacity int64) {
			if capacity <= 0 {
				fmt.Printf("%s: N/A\n", label)
				return
			}

			percentage := float64(usage) / float64(capacity) * 100
			fmt.Printf("%s: %.2f%%\n", label, percentage)
		}

		fmt.Println("\n--- Resource Utilization ---")

		printUtilization("CPU Request Utilization", cpuMilli, cpuRequest)
		printUtilization("CPU Limit Utilization", cpuMilli, cpuLimit)
		printUtilization("Memory Request Utilization", memoryBytes, memoryRequest)
		printUtilization("Memory Limit Utilization", memoryBytes, memoryLimit)
		fmt.Println("-----------------------------")

		// Evaluate pod health
		cpuHealth, cpuBasis := evaluatePodHealth(
			cpuMilli, cpuRequest, cpuLimit, CPUWarning, CPUCritical,
		)

		memoryHealth, memoryBasis := evaluatePodHealth(
			memoryBytes, memoryRequest, memoryLimit, MemoryWarning, MemoryCritical,
		)

		fmt.Println("\n--- Pod Health ---")
		fmt.Printf("CPU Health: %s (based on %s)\n", cpuHealth, cpuBasis)
		fmt.Printf("Memory Health: %s (based on %s)\n", memoryHealth, memoryBasis)
	}

}

func displayNodeInformation(
	node corev1.Node,
	metricsMap map[string]corev1.ResourceList,
) {
	fmt.Println("\n-----------------------------")
	fmt.Printf("Node Name: %s\n", node.Name)

	// Node readiness
	status := "NotReady"

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady &&
			condition.Status == corev1.ConditionTrue {
			status = "Ready"
		}
	}

	fmt.Printf("Node Status: %s\n", status)

	// Node capacity
	cpuCapacity := node.Status.Capacity.Cpu()
	memoryCapacity := node.Status.Capacity.Memory()

	fmt.Printf("CPU Capacity: %s\n", cpuCapacity.String())
	fmt.Printf("Memory Capacity: %s\n", memoryCapacity.String())

	// Live resource usage
	usage, exists := metricsMap[node.Name]

	if !exists {
		fmt.Println("Resource metrics: Not available")
		fmt.Println("-----------------------------")
		return
	}

	cpuUsage := usage.Cpu()
	memoryUsage := usage.Memory()

	fmt.Printf("CPU Usage: %s\n", cpuUsage.String())
	fmt.Printf("Memory Usage: %s\n", memoryUsage.String())

	// Calculate utilization percentages
	cpuPercent := float64(cpuUsage.MilliValue()) /
		float64(cpuCapacity.MilliValue()) * 100

	memoryPercent := float64(memoryUsage.Value()) /
		float64(memoryCapacity.Value()) * 100

	fmt.Printf("CPU Utilization: %.2f%%\n", cpuPercent)
	fmt.Printf("Memory Utilization: %.2f%%\n", memoryPercent)

	fmt.Println("-----------------------------")

	cpuStatus := evaluateThreshold(
		cpuPercent, CPUWarning, CPUCritical,
	)

	memoryStatus := evaluateThreshold(
		memoryPercent, MemoryWarning, MemoryCritical,
	)

	fmt.Printf("CPU Health: %s\n", cpuStatus)
	fmt.Printf("Memory Health: %s\n", memoryStatus)
}

func evaluateThreshold(value, warning, critical float64) string {
	if value >= critical {
		return "CRITICAL"
	}

	if value >= warning {
		return "WARNING"
	}

	return "HEALTHY"
}

func evaluatePodHealth(usage, request, limit int64, warning, critical float64) (string, string) {
	var capacity int64
	var basis string

	if limit > 0 {
		capacity = limit
		basis = "Limit"
	} else if request > 0 {
		capacity = request
		basis = "Request"
	} else {
		return "N/A", "No request or limit"
	}

	percentage := float64(usage) / float64(capacity) * 100

	return evaluateThreshold(percentage, warning, critical), basis
}
