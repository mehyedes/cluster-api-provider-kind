package util

import (
	"fmt"
	"net/url"
	"strconv"

	"k8s.io/client-go/tools/clientcmd"
	Kind "sigs.k8s.io/kind/pkg/cluster"
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

// AlreadyExists returns trye if a KIND cluster already exists
// Inspired from https://github.com/kubernetes-sigs/kind/blob/main/pkg/cluster/internal/create/create.go#L176
func AlreadyExists(p *Kind.Provider, name string) (bool, error) {
	n, err := p.ListNodes(name)
	if err != nil {
		return true, err
	}
	if len(n) != 0 {
		return true, nil
	}
	return false, nil
}
