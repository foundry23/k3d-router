package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirPrefersK3DRouterHomeOverEverything(t *testing.T) {
	t.Setenv("K3D_ROUTER_HOME", "/explicit/home")
	t.Setenv("XDG_CONFIG_HOME", "/some/xdg")

	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/explicit/home" {
		t.Fatalf("expected /explicit/home, got %s", got)
	}
}

func TestDirUsesXDGConfigHomeWhenNoExplicitHome(t *testing.T) {
	t.Setenv("K3D_ROUTER_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "/some/xdg")

	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/some/xdg", "k3d-router"); got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestDirFallsBackToHomeDotConfig(t *testing.T) {
	t.Setenv("K3D_ROUTER_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir available in this environment: %v", err)
	}

	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "k3d-router"); got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
