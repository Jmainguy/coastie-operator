package coastieservice

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPodsReadyTime(r *ReconcileCoastieService, name, namespace string, reqLogger logr.Logger, dsct string) {
	t, err := time.Parse(time.RFC3339, dsct)
	if err != nil {
		panic(err.Error())
	}

	opts := &client.ListOptions{}
	opts.SetLabelSelector(fmt.Sprintf("app=%s", name))
	opts.InNamespace(namespace)

	podList := &corev1.PodList{}
	ctx := context.TODO()
	r.client.List(ctx, opts, podList)

	for _, v := range podList.Items {
		for _, pv := range v.Status.Conditions {
			if pv.Type == "Ready" {
				timeToStart := pv.LastTransitionTime.Sub(t)
				reqLogger.Info("Pod Times", "Pod.Name", v.Name, "Pod.TimeToStartInSeconds", timeToStart, "NodeName", v.Spec.NodeName, "Namespace", namespace, "Name", name)
			}
		}
	}

	return
}

func getNodesWithoutPods(r *ReconcileCoastieService, name, namespace string) (nodes []string) {
	opts := &client.ListOptions{}
	nodesWithPods := make(map[string]string)
	nodeList := &corev1.NodeList{}
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	nodeList, _ = clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	for _, v := range nodeList.Items {
		nodesWithPods[v.Name] = "True"
	}

	opts.SetLabelSelector(fmt.Sprintf("app=%s", name))
	opts.InNamespace(namespace)

	podList := &corev1.PodList{}
	ctx := context.TODO()
	r.client.List(ctx, opts, podList)

	for _, v := range podList.Items {
		delete(nodesWithPods, v.Spec.NodeName)
	}

	for k, _ := range nodesWithPods {
		nodes = append(nodes, k)
	}
	return
}
