package engine

import (
	"context"
	"fmt"
	"log/slog"

	gsdkclient "github.com/docker/go-sdk/client"
	mobyclient "github.com/moby/moby/client"
)

type Client struct {
	sdk gsdkclient.SDKClient
}

func New(ctx context.Context) (*Client, error) {
	sdk, err := gsdkclient.New(ctx, gsdkclient.WithLogger(slog.Default()))
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	if _, err := sdk.Ping(ctx, mobyclient.PingOptions{}); err != nil {
		_ = sdk.Close()
		return nil, fmt.Errorf("docker daemon not reachable: %w", err)
	}
	slog.Debug("docker client ready")
	return &Client{sdk: sdk}, nil
}

func (c *Client) Close() error {
	return c.sdk.Close()
}
