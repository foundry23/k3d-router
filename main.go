package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/foundry23/k3d-router/internal/paths"
	"github.com/foundry23/k3d-router/internal/router"
)

//go:embed assets/traefik.yml
var traefikStatic []byte

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	defaultHome, _ := paths.Dir()

	settings := func(cmd *cli.Command) router.Settings {
		return router.NewSettings(
			cmd.String("image"),
			cmd.Int("http-port"),
			cmd.Int("https-port"),
			cmd.String("home"),
		)
	}

	app := &cli.Command{
		Name:                  "k3d-router",
		Usage:                 "a shared local reverse proxy for k3d clusters",
		Version:               version,
		EnableShellCompletion: true,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			setupLogging(cmd.String("log-level"))
			return ctx, nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "image", Value: "traefik:v3", Sources: cli.EnvVars("K3D_ROUTER_IMAGE"), Usage: "Traefik image"},
			&cli.IntFlag{Name: "http-port", Value: 80, Sources: cli.EnvVars("K3D_ROUTER_HTTP_PORT"), Usage: "host HTTP port"},
			&cli.IntFlag{Name: "https-port", Value: 443, Sources: cli.EnvVars("K3D_ROUTER_HTTPS_PORT"), Usage: "host HTTPS port"},
			&cli.StringFlag{Name: "home", Value: defaultHome, Sources: cli.EnvVars("K3D_ROUTER_HOME"), Usage: "config directory"},
			&cli.StringFlag{Name: "log-level", Value: "info", Sources: cli.EnvVars("K3D_ROUTER_LOG_LEVEL"), Usage: "log level: debug, info, warn, error"},
		},
		Commands: []*cli.Command{
			{
				Name:  "up",
				Usage: "Start the router and generate routing from all running k3d clusters",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return router.Reconcile(ctx, settings(cmd), traefikStatic, os.Stdout)
				},
			},
			{
				Name:  "reload",
				Usage: "Re-scan clusters and their Host CRDs and regenerate routing",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return router.Reconcile(ctx, settings(cmd), traefikStatic, os.Stdout)
				},
			},
			{
				Name:  "watch",
				Usage: "Reconcile continuously as k3d clusters start and stop (Ctrl-C to stop)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return router.Watch(ctx, settings(cmd), traefikStatic, os.Stdout)
				},
			},
			{
				Name:  "down",
				Usage: "Stop and remove the router container",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return router.Down(ctx, settings(cmd))
				},
			},
			{
				Name:  "status",
				Usage: "Show router status, attached networks, and current routes",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return router.Status(ctx, settings(cmd), os.Stdout)
				},
			},
			{
				Name:  "logs",
				Usage: "Follow the router (Traefik) logs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return router.Logs(ctx, settings(cmd))
				},
			},
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
}

func setupLogging(level string) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(level)})))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
