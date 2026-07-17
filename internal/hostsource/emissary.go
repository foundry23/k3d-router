package hostsource

import (
	"context"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var emissaryHostGVR = schema.GroupVersionResource{
	Group:    "getambassador.io",
	Version:  "v3alpha1",
	Resource: "hosts",
}

type Emissary struct{}

var _ Source = Emissary{}

func (Emissary) Name() string { return "emissary" }

func (Emissary) Hostnames(ctx context.Context, kubeContext string) []string {
	client, err := dynamicClient(kubeContext)
	if err != nil {
		slog.Debug("kube client unavailable", "source", "emissary", "context", kubeContext, "err", err)
		return nil
	}
	list, err := client.Resource(emissaryHostGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("no Host CRDs", "source", "emissary", "context", kubeContext, "err", err)
		return nil
	}
	return hostnamesFromItems(list.Items)
}

func hostnamesFromItems(items []unstructured.Unstructured) []string {
	var hostnames []string
	for _, item := range items {
		hostname, found, err := unstructured.NestedString(item.Object, "spec", "hostname")
		if err != nil || !found || hostname == "" {
			continue
		}
		hostnames = append(hostnames, hostname)
	}
	return hostnames
}
