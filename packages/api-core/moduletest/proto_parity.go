// Package moduletest provides shared assertions for scenario module registries.
package moduletest

import (
	"context"
	"fmt"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/module"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoFile describes the RPC-bearing portion of a module registry entry.
// Services may restrict parity checks to a subset of the file's services.
type ProtoFile struct {
	Module   string
	File     protoreflect.FileDescriptor
	Services []protoreflect.Name
}

// AssertRegistryContracts runs the shared invariants expected of a scenario
// module registry. The callbacks let each owner continue to expose its own
// registry without introducing a dependency from API Core into that scenario.
func AssertRegistryContracts(
	t *testing.T,
	allEndpoints func() []module.EndpointDescriptor,
	allSchemas func() []database.SchemaProvider,
	systemSchema func() string,
	protoFiles func() []ProtoFile,
	requireProtoFiles bool,
) {
	t.Helper()
	t.Run("endpoints_non_empty", func(t *testing.T) {
		if got := allEndpoints(); len(got) == 0 {
			t.Fatal("AllEndpoints must include at least the health endpoint")
		}
	})
	t.Run("endpoints_stable_order", func(t *testing.T) {
		a, b := allEndpoints(), allEndpoints()
		if len(a) != len(b) {
			t.Fatalf("non-deterministic endpoint count: %d then %d", len(a), len(b))
		}
		for i := range a {
			if a[i].ID != b[i].ID {
				t.Errorf("endpoint order changed at index %d: %q then %q", i, a[i].ID, b[i].ID)
			}
		}
	})
	t.Run("schemas_non_empty", func(t *testing.T) {
		got := allSchemas()
		if len(got) == 0 {
			t.Fatal("AllSchemas must include the system provider")
		}
		for i, provider := range got {
			if provider == nil {
				t.Errorf("AllSchemas[%d] is nil", i)
			}
		}
	})
	t.Run("system_schema_first", func(t *testing.T) {
		got := allSchemas()
		if len(got) == 0 {
			t.Fatal("AllSchemas returned no entries")
		}
		if got[0].Schema() != systemSchema() {
			t.Error("system home must be first in AllSchemas")
		}
	})
	t.Run("schemas_apply_idempotently", func(t *testing.T) {
		d := databasetest.NewSQLite(t)
		providers := allSchemas()
		if err := database.EnsureSchemas(context.Background(), d, providers...); err != nil {
			t.Fatalf("first schema apply: %v", err)
		}
		if err := database.EnsureSchemas(context.Background(), d, providers...); err != nil {
			t.Fatalf("second schema apply: %v", err)
		}
	})
	t.Run("proto_connect_parity", func(t *testing.T) {
		AssertProtoConnectParity(t, allEndpoints(), protoFiles(), requireProtoFiles)
	})
}

// AssertProtoConnectParity checks that each selected RPC has exactly one
// matching endpoint descriptor. requireFiles is false for registries that
// have not yet added a Connect-mounted product domain.
func AssertProtoConnectParity(t *testing.T, endpoints []module.EndpointDescriptor, files []ProtoFile, requireFiles bool) {
	t.Helper()
	if requireFiles && len(files) == 0 {
		t.Fatal("AllProtoFiles() returned no entries; every Connect-mounted module must be registered there")
	}

	byPath := make(map[string]int, len(endpoints))
	for _, endpoint := range endpoints {
		byPath[endpoint.Path]++
	}

	for _, entry := range files {
		services := entry.File.Services()
		if services.Len() == 0 {
			t.Errorf("module %q: proto file declares no services", entry.Module)
			continue
		}

		for s := 0; s < services.Len(); s++ {
			svc := services.Get(s)
			if len(entry.Services) > 0 && !containsService(entry.Services, svc.Name()) {
				continue
			}
			methods := svc.Methods()
			for m := 0; m < methods.Len(); m++ {
				method := methods.Get(m)
				wantPath := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
				count := byPath[wantPath]
				if count != 1 {
					t.Errorf("module %q: proto method %s.%s (expected EndpointDescriptor.Path %q) must have exactly one matching entry in AllEndpoints(); found %d",
						entry.Module, svc.FullName(), method.Name(), wantPath, count)
				}
			}
		}
	}
}

func containsService(services []protoreflect.Name, want protoreflect.Name) bool {
	for _, service := range services {
		if service == want {
			return true
		}
	}
	return false
}
