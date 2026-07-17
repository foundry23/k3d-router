package hostsource

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func dynamicClient(kubeContext string) (dynamic.Interface, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}
