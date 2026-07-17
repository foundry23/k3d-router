package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	mobyclient "github.com/moby/moby/client"
)

type Cluster struct {
	Name                  string
	Running               bool
	LoadBalancerContainer string
	Network               string
	Context               string
}

func (c *Client) ListClusters(ctx context.Context) ([]Cluster, error) {
	filters := mobyclient.Filters{}.
		Add("label", "app=k3d").
		Add("label", "k3d.role=loadbalancer")

	result, err := c.sdk.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("list k3d loadbalancer containers: %w", err)
	}

	clusters := make([]Cluster, 0, len(result.Items))
	for _, item := range result.Items {
		name := item.Labels["k3d.cluster"]
		if name == "" {
			continue
		}
		clusters = append(clusters, Cluster{
			Name:                  name,
			Running:               item.State == "running",
			LoadBalancerContainer: container(item.Names),
			Network:               "k3d-" + name,
			Context:               "k3d-" + name,
		})
	}
	slog.Debug("listed k3d clusters", "count", len(clusters))
	return clusters, nil
}

func container(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

func (c *Client) ClusterEvents(ctx context.Context) (<-chan struct{}, <-chan error) {
	filters := mobyclient.Filters{}.
		Add("type", "container").
		Add("event", "start", "die").
		Add("label", "k3d.role=loadbalancer")

	stream := c.sdk.Events(ctx, mobyclient.EventsListOptions{Filters: filters})
	slog.Debug("subscribed to docker cluster events")

	signals := make(chan struct{})
	errs := make(chan error, 1)
	go func() {
		defer close(signals)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-stream.Messages:
				if !ok {
					return
				}
				verb := string(msg.Action)
				switch msg.Action {
				case "start":
					verb = "started"
				case "die":
					verb = "died"
				}
				slog.Info("cluster loadbalancer "+verb,
					"container", msg.Actor.Attributes["name"],
					"cluster", msg.Actor.Attributes["k3d.cluster"])
				select {
				case signals <- struct{}{}:
				case <-ctx.Done():
					return
				}
			case err, ok := <-stream.Err:
				if ok && err != nil {
					errs <- err
				}
				return
			}
		}
	}()
	return signals, errs
}
