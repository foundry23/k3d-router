package hostsource

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestHostnamesFromItems(t *testing.T) {
	items := []unstructured.Unstructured{
		{Object: map[string]any{"spec": map[string]any{"hostname": "a.test"}}},
		{Object: map[string]any{"spec": map[string]any{"hostname": "b.test"}}},
		{Object: map[string]any{"spec": map[string]any{}}},               // no hostname
		{Object: map[string]any{"spec": map[string]any{"hostname": ""}}}, // empty hostname
	}
	got := hostnamesFromItems(items)
	want := []string{"a.test", "b.test"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
