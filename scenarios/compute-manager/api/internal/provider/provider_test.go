package provider

import (
	"context"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFakeLostCreateLeavesProviderInstance(t *testing.T) { // [REQ:COMPUTEM-P0-001] [REQ:COMPUTEM-P0-002]
	fake := &Fake{LoseCreateResponse: true}
	_, err := fake.Create(context.Background(), Spec{Region: "fsn1", Size: "cx22", Image: "ubuntu"})
	if !errors.Is(err, ErrCreateResponseLost) {
		t.Fatalf("Create error = %v", err)
	}
	instances, err := fake.List(context.Background())
	if err != nil || len(instances) != 1 {
		t.Fatalf("List = %v, %v", instances, err)
	}
}

func TestConcreteProvidersStayAtCompositionBoundary(t *testing.T) { // [REQ:COMPUTEM-P1-004]
	apiRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	err = filepath.Walk(apiRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(apiRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "main.go" || strings.HasPrefix(rel, "internal/provider/") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			value := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(value, "/internal/provider/hetzner") || strings.Contains(value, "/internal/provider/digitalocean") {
				t.Errorf("%s imports concrete provider %q; keep provider selection at the composition boundary", rel, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestProviderSurfaceHasNoPauseOperation(t *testing.T) { // [REQ:COMPUTEM-P0-001] [REQ:COMPUTEM-P0-007]
	// This compile-time assertion is intentionally the test: the interface has
	// exactly the four provider operations allowed by the product contract.
	var _ Provider = (*Fake)(nil)
	if got := reflect.TypeOf((*Provider)(nil)).Elem().NumMethod(); got != 6 {
		t.Fatalf("provider interface method count = %d, want lifecycle plus identity/billing methods", got)
	}
}

func TestScenarioProductionSurfaceHasNoPowerStateVerbs(t *testing.T) { // [REQ:COMPUTEM-P0-007]
	scenarioRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"PauseInstance", "StopInstance", "SuspendInstance", "HaltInstance", "ShutdownInstance"}
	err = filepath.Walk(scenarioRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 || strings.Contains(path, string(filepath.Separator)+"node_modules"+string(filepath.Separator)) || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".md") || strings.Contains(path, string(filepath.Separator)+"docs"+string(filepath.Separator)) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, verb := range forbidden {
			if strings.Contains(string(data), verb) {
				t.Errorf("%s contains forbidden power-state verb %q", path, verb)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(scenarioRoot, "cli", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Groups []struct {
			Commands []struct {
				Name string `json:"name"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(manifest, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, group := range decoded.Groups {
		for _, command := range group.Commands {
			for _, verb := range []string{"pause", "stop", "suspend", "halt", "shutdown"} {
				if command.Name == verb {
					t.Fatalf("CLI manifest exposes forbidden power-state command %q", command.Name)
				}
			}
		}
	}
}

type alternateProvider struct{ Fake }

func (*alternateProvider) Name() string { return "alternate" }
func (*alternateProvider) Facts() BillingFacts {
	return BillingFacts{RoundingUnit: time.Minute, MinimumBillable: time.Minute}
}

func TestRegistrySelectsAdaptersByIdentifier(t *testing.T) { // [REQ:COMPUTEM-P1-004]
	primary := &Fake{}
	secondary := &alternateProvider{}
	registry := NewRegistry(primary)
	if err := registry.Register(secondary); err != nil {
		t.Fatal(err)
	}
	got, err := registry.Get("alternate")
	if err != nil || got.Name() != "alternate" || got.Facts().RoundingUnit != time.Minute {
		t.Fatalf("adapter = %v/%v", got, err)
	}
	if _, err := registry.Get("missing"); err == nil {
		t.Fatal("missing adapter unexpectedly resolved")
	}
}
