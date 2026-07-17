package hostsource

import (
	"context"
	"reflect"
	"testing"
)

type fakeSource struct {
	name  string
	hosts []string
}

func (f fakeSource) Name() string                               { return f.name }
func (f fakeSource) Hostnames(context.Context, string) []string { return f.hosts }

func TestDiscoverUnionsAndDedups(t *testing.T) {
	sources := []Source{
		fakeSource{name: "a", hosts: []string{"x.test", "y.test"}},
		fakeSource{name: "b", hosts: []string{"y.test", "z.test", ""}},
	}
	got := Discover(context.Background(), "ctx", sources)
	want := []string{"x.test", "y.test", "z.test"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDiscoverNoHosts(t *testing.T) {
	got := Discover(context.Background(), "ctx", []Source{fakeSource{name: "empty"}})
	if len(got) != 0 {
		t.Fatalf("expected no hostnames, got %v", got)
	}
}
