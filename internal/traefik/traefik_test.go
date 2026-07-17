package traefik

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildConfigEmpty(t *testing.T) {
	cfg, warnings := BuildConfig(nil)
	if cfg.HTTP != nil || cfg.TCP != nil {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "{}" {
		t.Fatalf("expected {}, got %s", data)
	}
}

func TestBuildConfigSingleClusterSortsAndRoutes(t *testing.T) {
	routes := []Route{{
		Cluster:               "jobilla",
		LoadBalancerContainer: "k3d-jobilla-serverlb",
		Hostnames:             []string{"b.jobilla.test", "a.jobilla.test"},
	}}
	cfg, warnings := BuildConfig(routes)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if cfg.HTTP == nil || cfg.TCP == nil {
		t.Fatal("expected http and tcp sections")
	}
	if got := cfg.HTTP.Routers["jobilla"].Rule; got != "Host(`a.jobilla.test`) || Host(`b.jobilla.test`)" {
		t.Fatalf("unexpected http rule: %s", got)
	}
	tcp := cfg.TCP.Routers["jobilla"]
	if tcp.Rule != "HostSNI(`a.jobilla.test`) || HostSNI(`b.jobilla.test`)" {
		t.Fatalf("unexpected tcp rule: %s", tcp.Rule)
	}
	if !tcp.TLS.Passthrough {
		t.Fatal("expected tcp TLS passthrough")
	}
	if got := cfg.HTTP.Services["jobilla"].LoadBalancer.Servers[0].URL; got != "http://k3d-jobilla-serverlb:80" {
		t.Fatalf("unexpected http backend: %s", got)
	}
	if got := cfg.TCP.Services["jobilla"].LoadBalancer.Servers[0].Address; got != "k3d-jobilla-serverlb:443" {
		t.Fatalf("unexpected tcp backend: %s", got)
	}
}

func TestMarshalIsYAMLAndRoundTrips(t *testing.T) {
	cfg, _ := BuildConfig([]Route{
		{Cluster: "jobilla", LoadBalancerContainer: "k3d-jobilla-serverlb", Hostnames: []string{"a.test", "b.test"}},
	})
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, "rule:") || strings.Contains(out, `"rule"`) {
		t.Fatalf("expected YAML output (unquoted keys), got:\n%s", out)
	}

	back, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(back.Hosts(), cfg.Hosts()) {
		t.Fatalf("round-trip host mismatch: %v vs %v", back.Hosts(), cfg.Hosts())
	}
	if !back.TCP.Routers["jobilla"].TLS.Passthrough {
		t.Fatal("tcp TLS passthrough lost in round-trip")
	}
}

func TestHostsExtractsFromRules(t *testing.T) {
	cfg, _ := BuildConfig([]Route{
		{Cluster: "jobilla", LoadBalancerContainer: "k3d-jobilla-serverlb", Hostnames: []string{"a.test", "b.test"}},
	})
	got := cfg.Hosts()
	if len(got) != 2 || got["a.test"] != "jobilla" || got["b.test"] != "jobilla" {
		t.Fatalf("unexpected host map: %v", got)
	}
	if len((Config{}).Hosts()) != 0 {
		t.Fatal("empty config should yield no routes")
	}
}

func TestDiffAddRemoveMove(t *testing.T) {
	oldRoutes := map[string]string{"keep.test": "a", "gone.test": "a", "move.test": "a"}
	newRoutes := map[string]string{"keep.test": "a", "add.test": "b", "move.test": "b"}
	added, removed := Diff(oldRoutes, newRoutes)

	// added (sorted): add.test→b, move.test→b (moved from a)
	if len(added) != 2 || added[0].Host != "add.test" || added[1].Host != "move.test" || added[1].Cluster != "b" {
		t.Fatalf("unexpected added: %v", added)
	}
	// removed (sorted): gone.test→a, move.test→a (its old cluster)
	if len(removed) != 2 || removed[0].Host != "gone.test" || removed[1].Host != "move.test" || removed[1].Cluster != "a" {
		t.Fatalf("unexpected removed: %v", removed)
	}
}

func TestDiffNoChange(t *testing.T) {
	routes := map[string]string{"x.test": "a"}
	added, removed := Diff(routes, map[string]string{"x.test": "a"})
	if len(added) != 0 || len(removed) != 0 {
		t.Fatalf("expected no changes, got added=%v removed=%v", added, removed)
	}
}

func TestBuildConfigCollisionFirstClusterWins(t *testing.T) {
	routes := []Route{
		{Cluster: "a", LoadBalancerContainer: "k3d-a-serverlb", Hostnames: []string{"shared.test"}},
		{Cluster: "b", LoadBalancerContainer: "k3d-b-serverlb", Hostnames: []string{"shared.test", "b-only.test"}},
	}
	cfg, warnings := BuildConfig(routes)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 collision warning, got %v", warnings)
	}
	if _, ok := cfg.HTTP.Routers["a"]; !ok {
		t.Fatal("expected a to keep shared.test")
	}
	if got := cfg.HTTP.Routers["b"].Rule; got != "Host(`b-only.test`)" {
		t.Fatalf("expected b to route only b-only.test, got %s", got)
	}
}
