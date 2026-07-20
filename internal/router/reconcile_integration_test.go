//go:build integration

package router

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/foundry23/k3d-router/internal/engine"
	"github.com/foundry23/k3d-router/internal/traefik"
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

func TestReconcileAgainstK3d(t *testing.T) {
	cluster := os.Getenv("IT_K3D_CLUSTER")
	hostname := os.Getenv("IT_K3D_HOSTNAME")
	if cluster == "" || hostname == "" {
		t.Skip("set IT_K3D_CLUSTER and IT_K3D_HOSTNAME to run the k3d integration test")
	}

	ctx := context.Background()
	dir := t.TempDir()

	s := Settings{
		Container: "k3d-router-it-router",
		Network:   "k3d-router-it-router",
		Image:     "traefik:v3",
		HTTPPort:  18081,
		HTTPSPort: 18444,
		ConfigDir: dir,
	}
	t.Cleanup(func() { _ = Down(ctx, s) })

	if err := Reconcile(ctx, s, []byte(itStatic), io.Discard); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "dynamic", "routes.yml"))
	if err != nil {
		t.Fatalf("read routes.yml: %v", err)
	}
	cfg, err := traefik.Parse(data)
	if err != nil {
		t.Fatalf("parse routes.yml: %v", err)
	}
	if got := cfg.Hosts()[hostname]; got != cluster {
		t.Fatalf("routes.yml: expected %q -> %q, got hosts=%v", hostname, cluster, cfg.Hosts())
	}

	cli, err := engine.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cli.Close() }()
	info, exists, err := cli.InspectContainer(ctx, s.Container)
	if err != nil || !exists || !info.Running {
		t.Fatalf("router container not running: exists=%v err=%v", exists, err)
	}
}
