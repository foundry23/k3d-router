package router

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/foundry23/k3d-router/internal/engine"
	"github.com/foundry23/k3d-router/internal/hostsource"
	"github.com/foundry23/k3d-router/internal/traefik"
)

const (
	containerName = "k3d-router"
	networkName   = "k3d-router"
)

type Settings struct {
	Container string
	Network   string
	Image     string
	HTTPPort  int
	HTTPSPort int
	ConfigDir string
}

func NewSettings(image string, httpPort, httpsPort int, configDir string) Settings {
	return Settings{
		Container: containerName,
		Network:   networkName,
		Image:     image,
		HTTPPort:  httpPort,
		HTTPSPort: httpsPort,
		ConfigDir: configDir,
	}
}

func Reconcile(ctx context.Context, s Settings, staticConfig []byte, w io.Writer) error {
	cli, err := engine.New(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	dynamicDir := filepath.Join(s.ConfigDir, "dynamic")
	if err := os.MkdirAll(dynamicDir, 0o755); err != nil {
		return err
	}
	routesPath := filepath.Join(dynamicDir, "routes.yml")

	clusters, err := cli.ListClusters(ctx)
	if err != nil {
		return err
	}
	sources := hostsource.Default()
	var routes []traefik.Route
	var connectNetworks []string
	for _, cluster := range clusters {
		if !cluster.Running {
			slog.Debug("cluster loadbalancer not running — skipping", "cluster", cluster.Name)
			continue
		}
		hostnames := hostsource.Discover(ctx, cluster.Context, sources)
		if len(hostnames) == 0 {
			slog.Debug("cluster has no Host CRDs — no routes", "cluster", cluster.Name)
			continue
		}
		routes = append(routes, traefik.Route{
			Cluster:               cluster.Name,
			LoadBalancerContainer: cluster.LoadBalancerContainer,
			Hostnames:             hostnames,
		})
		connectNetworks = append(connectNetworks, cluster.Network)
		slog.Debug("cluster routes discovered", "cluster", cluster.Name, "hosts", strings.Join(hostnames, " "))
	}

	cfg, warnings := traefik.BuildConfig(routes)
	for _, warning := range warnings {
		slog.Warn(warning)
	}

	oldHosts := map[string]string{}
	if existing, err := os.ReadFile(routesPath); err == nil {
		if oldConfig, err := traefik.Parse(existing); err == nil {
			oldHosts = oldConfig.Hosts()
		}
	}
	added, removed := traefik.Diff(oldHosts, cfg.Hosts())

	data, err := cfg.Marshal()
	if err != nil {
		return err
	}
	staticChanged, err := writeIfChanged(filepath.Join(s.ConfigDir, "traefik.yml"), staticConfig)
	if err != nil {
		return err
	}
	dynamicChanged, err := writeIfChanged(routesPath, append(data, '\n'))
	if err != nil {
		return err
	}

	if err := cli.EnsureNetwork(ctx, s.Network); err != nil {
		return err
	}
	info, exists, err := cli.InspectContainer(ctx, s.Container)
	if err != nil {
		return err
	}
	loadedFresh := false
	switch {
	case !exists:
		fmt.Fprintf(w, "Starting %s (%s) on :%d / :%d ...\n", s.Container, s.Image, s.HTTPPort, s.HTTPSPort)
		if err := cli.RunRouter(ctx, engine.RunOpts{
			Name:      s.Container,
			Network:   s.Network,
			Image:     s.Image,
			HTTPPort:  s.HTTPPort,
			HTTPSPort: s.HTTPSPort,
			ConfigDir: s.ConfigDir,
		}); err != nil {
			return err
		}
		loadedFresh = true
	case !info.Running:
		if err := cli.StartContainer(ctx, s.Container); err != nil {
			return err
		}
		loadedFresh = true
	}

	for _, network := range connectNetworks {
		if err := cli.ConnectNetwork(ctx, network, s.Container); err != nil {
			return err
		}
	}

	if !loadedFresh && (staticChanged || dynamicChanged) {
		slog.Debug("config changed — restarting router to reload")
		if err := cli.RestartContainer(ctx, s.Container); err != nil {
			return err
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		fmt.Fprintln(w, "No routing changes.")
		return nil
	}
	for _, change := range removed {
		fmt.Fprintf(w, "- %s → %s\n", change.Host, change.Cluster)
	}
	for _, change := range added {
		fmt.Fprintf(w, "+ %s → %s\n", change.Host, change.Cluster)
	}
	return nil
}

func writeIfChanged(path string, data []byte) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return false, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func Down(ctx context.Context, s Settings) error {
	cli, err := engine.New(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()
	return cli.RemoveContainer(ctx, s.Container)
}

func Status(ctx context.Context, s Settings, w io.Writer) error {
	cli, err := engine.New(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	info, exists, err := cli.InspectContainer(ctx, s.Container)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintf(w, "container: %s (not created — run 'k3d-router up')\n", s.Container)
	} else {
		state := "stopped"
		if info.Running {
			state = "running"
		}
		networks := append([]string(nil), info.Networks...)
		sort.Strings(networks)
		fmt.Fprintf(w, "container: %s (%s)\n", s.Container, state)
		fmt.Fprintf(w, "networks:  %s\n", strings.Join(networks, ", "))
	}

	fmt.Fprintln(w, "\n--- routes ---")
	data, err := os.ReadFile(filepath.Join(s.ConfigDir, "dynamic", "routes.yml"))
	if err != nil {
		fmt.Fprintln(w, "(none generated yet)")
		return nil
	}
	_, err = w.Write(data)
	return err
}

func Logs(ctx context.Context, s Settings) error {
	cli, err := engine.New(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()
	return cli.Logs(ctx, s.Container)
}

func Watch(ctx context.Context, s Settings, staticConfig []byte, w io.Writer) error {
	if err := Reconcile(ctx, s, staticConfig, w); err != nil {
		return err
	}

	cli, err := engine.New(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	signals, errs := cli.ClusterEvents(ctx)
	fmt.Fprintln(w, "Watching for cluster changes (Ctrl-C to stop)...")

	const debounce = time.Second
	timer := time.NewTimer(debounce)
	timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			return fmt.Errorf("docker event stream: %w", err)
		case _, ok := <-signals:
			if !ok {
				return nil
			}
			slog.Debug("cluster event received, debouncing")
			timer.Reset(debounce)
		case <-timer.C:
			fmt.Fprintln(w, "cluster change detected — reconciling")
			if err := Reconcile(ctx, s, staticConfig, w); err != nil {
				slog.Error("reconcile failed", "err", err)
			}
		}
	}
}
