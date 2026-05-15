package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// BuildClient returns a Kubernetes clientset. Tries in-cluster config first;
// falls back to the provided kubeconfig path, then to ~/.kube/config.
func BuildClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		path := kubeconfigPath
		if path == "" {
			if home := homedir.HomeDir(); home != "" {
				path = filepath.Join(home, ".kube", "config")
			}
		}
		if path == "" {
			return nil, fmt.Errorf("no in-cluster config and no kubeconfig path available")
		}
		if _, statErr := os.Stat(path); statErr != nil {
			return nil, fmt.Errorf("kubeconfig %q not found: %w (in-cluster err: %v)", path, statErr, err)
		}
		cfg, err = clientcmd.BuildConfigFromFlags("", path)
		if err != nil {
			return nil, fmt.Errorf("building kubeconfig: %w", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}
