package resources

import (
	"context"
	"reflect"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestManagedServiceBackendArgumentsFollowSelectedBackend(t *testing.T) {
	manifest, err := manifestpkg.Load("../../resources/whisper/resource.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_GPU", "off")
	got, err := managedServiceBackendArguments(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"--no-gpu"}) {
		t.Fatalf("CPU backend argv = %#v", got)
	}
	t.Setenv("VROOLI_GPU", "on")
	got, err = managedServiceBackendArguments(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("Vulkan backend inherited CPU argv: %#v", got)
	}
}
