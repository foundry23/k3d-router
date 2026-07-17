package engine

import (
	"context"
	"log/slog"
	"slices"

	"github.com/containerd/errdefs"
	mobyclient "github.com/moby/moby/client"
)

func (c *Client) EnsureNetwork(ctx context.Context, name string) error {
	if _, err := c.sdk.NetworkInspect(ctx, name, mobyclient.NetworkInspectOptions{}); err == nil {
		return nil
	} else if !errdefs.IsNotFound(err) {
		return err
	}
	slog.Debug("creating docker network", "network", name)
	_, err := c.sdk.NetworkCreate(ctx, name, mobyclient.NetworkCreateOptions{Driver: "bridge"})
	return err
}

func (c *Client) ConnectNetwork(ctx context.Context, network, container string) error {
	info, exists, err := c.InspectContainer(ctx, container)
	if err != nil {
		return err
	}
	if exists && slices.Contains(info.Networks, network) {
		slog.Debug("router already attached to cluster network", "network", network)
		return nil
	}
	slog.Debug("attaching router to cluster network", "network", network)
	_, err = c.sdk.NetworkConnect(ctx, network, mobyclient.NetworkConnectOptions{Container: container})
	return err
}
