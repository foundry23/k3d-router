//go:build integration

package engine

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	mobyclient "github.com/moby/moby/client"
)

const itStatic = `entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: false
`

func TestEngineDockerLifecycle(t *testing.T) {
	const (
		container = "k3d-router-it-engine"
		network   = "k3d-router-it-engine-net"
		extraNet  = "k3d-router-it-engine-extra"
		image     = "traefik:v3"
	)

	ctx := context.Background()
	cli, err := New(ctx)
	if err != nil {
		t.Fatalf("docker not available: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	teardown := func() {
		_, _ = cli.sdk.ContainerRemove(ctx, container, mobyclient.ContainerRemoveOptions{Force: true})
		_, _ = cli.sdk.NetworkRemove(ctx, network, mobyclient.NetworkRemoveOptions{})
		_, _ = cli.sdk.NetworkRemove(ctx, extraNet, mobyclient.NetworkRemoveOptions{})
	}
	teardown()
	t.Cleanup(teardown)

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "dynamic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "traefik.yml"), []byte(itStatic), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cli.EnsureNetwork(ctx, network); err != nil {
		t.Fatalf("EnsureNetwork: %v", err)
	}
	if err := cli.EnsureNetwork(ctx, network); err != nil {
		t.Fatalf("EnsureNetwork is not idempotent: %v", err)
	}

	if err := cli.RunRouter(ctx, RunOpts{
		Name: container, Network: network, Image: image,
		HTTPPort: 18080, HTTPSPort: 18443, ConfigDir: dir,
	}); err != nil {
		t.Fatalf("RunRouter: %v", err)
	}

	info, exists, err := cli.InspectContainer(ctx, container)
	if err != nil || !exists {
		t.Fatalf("InspectContainer after run: exists=%v err=%v", exists, err)
	}
	if !info.Running {
		t.Fatal("router container should be running")
	}
	if !slices.Contains(info.Networks, network) {
		t.Fatalf("expected attachment to %q, got %v", network, info.Networks)
	}

	if err := cli.EnsureNetwork(ctx, extraNet); err != nil {
		t.Fatalf("EnsureNetwork extra: %v", err)
	}
	if err := cli.ConnectNetwork(ctx, extraNet, container); err != nil {
		t.Fatalf("ConnectNetwork: %v", err)
	}
	if err := cli.ConnectNetwork(ctx, extraNet, container); err != nil {
		t.Fatalf("ConnectNetwork is not idempotent when already attached: %v", err)
	}
	if info, _, _ = cli.InspectContainer(ctx, container); !slices.Contains(info.Networks, extraNet) {
		t.Fatalf("expected attachment to %q, got %v", extraNet, info.Networks)
	}

	if err := cli.RemoveContainer(ctx, container); err != nil {
		t.Fatalf("RemoveContainer: %v", err)
	}
	if _, exists, _ := cli.InspectContainer(ctx, container); exists {
		t.Fatal("container should be gone after RemoveContainer")
	}
	if err := cli.RemoveContainer(ctx, container); err != nil {
		t.Fatalf("RemoveContainer should treat missing as success: %v", err)
	}
}
