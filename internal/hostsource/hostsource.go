package hostsource

import (
	"context"
	"log/slog"
)

type Source interface {
	Name() string
	Hostnames(ctx context.Context, kubeContext string) []string
}

func Default() []Source {
	return []Source{Emissary{}}
}

func Discover(ctx context.Context, kubeContext string, sources []Source) []string {
	seen := map[string]bool{}
	var hostnames []string
	for _, source := range sources {
		found := source.Hostnames(ctx, kubeContext)
		slog.Debug("host source queried", "source", source.Name(), "context", kubeContext, "found", len(found))
		for _, host := range found {
			if host == "" || seen[host] {
				continue
			}
			seen[host] = true
			hostnames = append(hostnames, host)
		}
	}
	return hostnames
}
