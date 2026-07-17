package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/containerd/errdefs"
	gsdkcontainer "github.com/docker/go-sdk/container"
	"github.com/moby/moby/api/pkg/stdcopy"
	apicontainer "github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
)

type ContainerInfo struct {
	Running  bool
	Networks []string
}

func (c *Client) InspectContainer(ctx context.Context, name string) (*ContainerInfo, bool, error) {
	res, err := c.sdk.ContainerInspect(ctx, name, mobyclient.ContainerInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	info := &ContainerInfo{}
	if res.Container.State != nil {
		info.Running = res.Container.State.Running
	}
	if res.Container.NetworkSettings != nil {
		for networkName := range res.Container.NetworkSettings.Networks {
			info.Networks = append(info.Networks, networkName)
		}
	}
	return info, true, nil
}

type RunOpts struct {
	Name      string
	Network   string
	Image     string
	HTTPPort  int
	HTTPSPort int
	ConfigDir string
}

func (c *Client) RunRouter(ctx context.Context, o RunOpts) error {
	slog.Debug("creating router container", "image", o.Image, "http_port", o.HTTPPort, "https_port", o.HTTPSPort)
	_, err := gsdkcontainer.Run(
		ctx,
		gsdkcontainer.WithClient(c.sdk),
		gsdkcontainer.WithImage(o.Image),
		gsdkcontainer.WithName(o.Name),
		gsdkcontainer.WithLabels(map[string]string{"app": "k3d-router"}),
		gsdkcontainer.WithNetworkName(nil, o.Network),
		gsdkcontainer.WithExposedPorts(
			fmt.Sprintf("%d:80/tcp", o.HTTPPort),
			fmt.Sprintf("%d:443/tcp", o.HTTPSPort),
		),
		gsdkcontainer.WithHostConfigModifier(func(hostConfig *apicontainer.HostConfig) {
			hostConfig.RestartPolicy = apicontainer.RestartPolicy{Name: apicontainer.RestartPolicyUnlessStopped}
			hostConfig.Binds = append(
				hostConfig.Binds,
				fmt.Sprintf("%s/traefik.yml:/etc/traefik/traefik.yml:ro", o.ConfigDir),
				fmt.Sprintf("%s/dynamic:/etc/traefik/dynamic:ro", o.ConfigDir),
			)
		}),
	)
	return err
}

func (c *Client) StartContainer(ctx context.Context, name string) error {
	_, err := c.sdk.ContainerStart(ctx, name, mobyclient.ContainerStartOptions{})
	return err
}

func (c *Client) RemoveContainer(ctx context.Context, name string) error {
	_, err := c.sdk.ContainerRemove(ctx, name, mobyclient.ContainerRemoveOptions{Force: true})
	if err != nil && errdefs.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) RestartContainer(ctx context.Context, name string) error {
	_, err := c.sdk.ContainerRestart(ctx, name, mobyclient.ContainerRestartOptions{})
	return err
}

func (c *Client) Logs(ctx context.Context, name string) error {
	reader, err := c.sdk.ContainerLogs(ctx, name, mobyclient.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
	return err
}
