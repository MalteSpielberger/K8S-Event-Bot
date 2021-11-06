package k8s

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

type KubernetesApi struct {
	clientSet *kubernetes.Clientset
}

// NewKubernetesApi creates a new k8s-api.
// The kube-config will be loaded from the
// standard path: $HOME/.kube/config
func NewKubernetesApi() (*KubernetesApi, error) {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("cannot build config for k8s: %w", err)
	}

	clientSet, err :=  kubernetes.NewForConfig(config)

	if err != nil {
		return nil, fmt.Errorf("cannot create new clientset: %w", err)
	}

	return &KubernetesApi{clientSet: clientSet}, nil
}

func (k *KubernetesApi) AppsV1() v1.AppsV1Interface {
	return k.clientSet.AppsV1()
}