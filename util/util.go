package util

import (
	"fmt"
	"net/url"
	"strconv"

	"k8s.io/client-go/tools/clientcmd"
)

// GetControlPlane returns the Hostname and port number of a KIND cluster from the provided kubeconfig file
func GetControlPlane(kubeconfigData string) (string, int32, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfigData))
	if err != nil {
		return "", 0, fmt.Errorf("failed while generating kubernetes client config: %w", err)
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return "", 0, fmt.Errorf("failed while getting kubenretes client config: %w", err)
	}

	u, err := url.Parse(restConfig.Host)
	if err != nil {
		return "", 0, fmt.Errorf("failed while parsing control plane endpoint URL: %w", err)
	}

	port, err := strconv.ParseInt(u.Port(), 10, 32)
	if err != nil {
		return "", 0, fmt.Errorf("failed while parsing control plane port number: %w", err)
	}
	return u.Hostname(), int32(port), nil
}
