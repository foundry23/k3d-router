package paths

import (
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	if home := os.Getenv("K3D_ROUTER_HOME"); home != "" {
		return home, nil
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "k3d-router"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "k3d-router"), nil
}
