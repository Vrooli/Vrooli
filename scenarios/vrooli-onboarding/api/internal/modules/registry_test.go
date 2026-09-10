package modules_test

import (
	"fmt"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/modules"
)

// TestProtoConnectParity is the global proto-to-descriptor drift gate. It is
// vacuously green during the REST-only foundation and becomes live as soon as
// a Connect domain registers its generated FileDescriptor.
func TestProtoConnectParity(t *testing.T) {
	if err := validateProtoConnectParity(modules.AllEndpoints(), modules.AllProtoFiles()); err != nil {
		t.Fatal(err)
	}
}

func TestProtoConnectParityHasTeeth(t *testing.T) {
	endpoints := modules.AllEndpoints()
	endpoints = endpoints[:len(endpoints)-1]
	if err := validateProtoConnectParity(endpoints, modules.AllProtoFiles()); err == nil {
		t.Fatal("expected a missing endpoint descriptor to fail proto parity")
	}
}

func validateProtoConnectParity(endpoints []module.EndpointDescriptor, protoFiles []modules.ProtoFileEntry) error {
	byPath := make(map[string]int, len(endpoints))
	for _, endpoint := range endpoints {
		byPath[endpoint.Path]++
	}
	for _, entry := range protoFiles {
		services := entry.File.Services()
		if services.Len() == 0 {
			return fmt.Errorf("module %q: proto file declares no services", entry.Module)
		}
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				method := service.Methods().Get(j)
				want := fmt.Sprintf("/%s/%s", service.FullName(), method.Name())
				if byPath[want] != 1 {
					return fmt.Errorf("module %q: missing or duplicate descriptor for %s", entry.Module, want)
				}
			}
		}
	}
	return nil
}
