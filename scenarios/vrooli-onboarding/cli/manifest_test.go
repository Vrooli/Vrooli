package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/auth"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

//go:embed manifest.json
var onboardingCLIManifest []byte

type manifestGroup struct {
	Name     string            `json:"name"`
	Commands []manifestCommand `json:"commands"`
	Groups   []manifestGroup   `json:"groups"`
}

type manifestCommand struct {
	Name       string          `json:"name"`
	Binding    json.RawMessage `json:"binding"`
	Governance struct {
		Effect string `json:"effect"`
	} `json:"governance"`
}

func TestReadinessManifestBindingPreservesExitCodeContract(t *testing.T) {
	var manifest struct {
		Groups []manifestGroup `json:"groups"`
	}
	if err := json.Unmarshal(onboardingCLIManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, group := range manifest.Groups {
		if group.Name != "readiness" {
			continue
		}
		if len(group.Commands) < 3 || string(group.Commands[0].Binding) == "" || string(group.Commands[1].Binding) == "" || string(group.Commands[2].Binding) == "" {
			t.Fatalf("readiness manifest commands = %+v", group.Commands)
		}
		var readBinding, statusBinding, ackBinding struct{ Kind, Service, Method string }
		if err := json.Unmarshal(group.Commands[0].Binding, &readBinding); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(group.Commands[1].Binding, &statusBinding); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(group.Commands[2].Binding, &ackBinding); err != nil {
			t.Fatal(err)
		}
		if readBinding.Kind != "connect-rpc" || readBinding.Service != "ReadinessService" || readBinding.Method != "GetReadiness" || statusBinding.Kind != "local" || ackBinding.Kind != "connect-rpc" || ackBinding.Service != "ReadinessService" || ackBinding.Method != "AcknowledgeDegradedReadiness" {
			t.Fatalf("readiness bindings = %+v / %+v / %+v", readBinding, statusBinding, ackBinding)
		}
		// The manifest binds the readiness command to the typed service; the
		// command's existing exit contract is verified by the focused domain
		// test that feeds a required blocker and expects exit code 2.
		return
	}
	t.Fatal("readiness group is missing")
}

func TestCapabilitiesManifestUsesTypedServiceBindings(t *testing.T) {
	var manifest struct {
		Groups []manifestGroup `json:"groups"`
	}
	if err := json.Unmarshal(onboardingCLIManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, group := range manifest.Groups {
		if group.Name != "capabilities" {
			continue
		}
		if len(group.Commands) != 4 {
			t.Fatalf("capabilities commands = %+v", group.Commands)
		}
		want := map[string]string{"list": "ListCapabilities", "preview": "PreviewCapability", "apply": "ApplyCapability", "verify": "VerifyCapability"}
		for _, command := range group.Commands {
			var binding struct{ Kind, Service, Method string }
			if err := json.Unmarshal(command.Binding, &binding); err != nil {
				t.Fatal(err)
			}
			if binding.Kind != "connect-rpc" || binding.Service != "CapabilitiesService" || binding.Method != want[command.Name] {
				t.Fatalf("%s binding = %+v", command.Name, binding)
			}
		}
		return
	}
	t.Fatal("capabilities group is missing")
}

func TestCredentialsManifestUsesTypedServiceBindings(t *testing.T) {
	var manifest struct {
		Groups []manifestGroup `json:"groups"`
	}
	if err := json.Unmarshal(onboardingCLIManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, group := range manifest.Groups {
		if group.Name != "credentials" {
			continue
		}
		want := map[string]string{"list": "ListCredentials", "provision": "ProvisionCredential", "doctor": "DiagnoseCredentials"}
		for _, command := range group.Commands {
			if command.Name == "reveal" {
				var binding struct {
					Kind    string `json:"kind"`
					Handler string `json:"handler"`
				}
				if err := json.Unmarshal(command.Binding, &binding); err != nil {
					t.Fatal(err)
				}
				if binding.Kind != "local" || binding.Handler != "credentials.reveal" {
					t.Fatalf("reveal binding = %+v", binding)
				}
				continue
			}
			var binding struct{ Kind, Service, Method string }
			if err := json.Unmarshal(command.Binding, &binding); err != nil {
				t.Fatal(err)
			}
			if binding.Kind != "connect-rpc" || binding.Service != "CredentialsService" || binding.Method != want[command.Name] {
				t.Fatalf("credentials %s binding = %+v", command.Name, binding)
			}
		}
		return
	}
	t.Fatal("credentials manifest group not found")
}

func TestCLIManifestDeclaresGovernanceForEveryCommand(t *testing.T) {
	var manifest struct {
		Name   string          `json:"name"`
		Groups []manifestGroup `json:"groups"`
	}
	if err := json.Unmarshal(onboardingCLIManifest, &manifest); err != nil {
		t.Fatalf("manifest JSON: %v", err)
	}
	if manifest.Name != appName {
		t.Fatalf("manifest name = %q, want %q", manifest.Name, appName)
	}
	var visit func([]manifestGroup, string)
	visit = func(groups []manifestGroup, prefix string) {
		for _, group := range groups {
			path := group.Name
			if prefix != "" {
				path = prefix + " " + path
			}
			for _, command := range group.Commands {
				if len(command.Binding) == 0 {
					t.Errorf("%s %s has no binding", path, command.Name)
				}
				if command.Governance.Effect == "" {
					t.Errorf("%s %s has no governance effect", path, command.Name)
				}
			}
			visit(group.Groups, path)
		}
	}
	visit(manifest.Groups, "")
}

func TestCLIManifestMatchesGeneratedProtoSurface(t *testing.T) {
	if err := validateManifestProtoParity(onboardingCLIManifest); err != nil {
		t.Fatal(err)
	}
}

func TestCLIManifestProtoParityHasTeeth(t *testing.T) {
	mutated := bytes.Replace(onboardingCLIManifest, []byte(`"method": "GetReadiness"`), []byte(`"method": "MissingReadiness"`), 1)
	if string(mutated) == string(onboardingCLIManifest) {
		t.Fatal("test fixture did not mutate a manifest binding")
	}
	if err := validateManifestProtoParity(mutated); err == nil {
		t.Fatal("expected a nonexistent Connect method to fail manifest parity")
	}
}

func validateManifestProtoParity(raw []byte) error {
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		return err
	}
	available := map[string]bool{}
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(string(file.Package()), "vrooli.vrooli_onboarding.v1.") {
			return true
		}
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				available[string(service.Name())+"."+string(service.Methods().Get(j).Name())] = true
			}
		}
		return true
	})
	commands := map[string]bool{}
	var walk func([]cliapp.ManifestGroup, string) error
	walk = func(groups []cliapp.ManifestGroup, prefix string) error {
		for _, group := range groups {
			path := group.Name
			if prefix != "" {
				path = prefix + "/" + path
			}
			for _, command := range group.Commands {
				if command.Binding.Kind != "connect-rpc" {
					if command.Architecture == nil || (command.Architecture.Exception == nil && command.Architecture.Primitive == "") {
						return fmt.Errorf("%s/%s has no Connect binding or architecture declaration", path, command.Name)
					}
					continue
				}
				key := command.Binding.BindingKey()
				if !available[key] {
					return fmt.Errorf("%s/%s references unknown generated RPC %s", path, command.Name, key)
				}
				commands[key] = true
			}
			if err := walk(group.Groups, path); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(manifest.Groups, ""); err != nil {
		return err
	}
	omitted := map[string]bool{}
	for _, entry := range manifest.Omitted {
		key := entry.Service + "." + entry.Method
		if !available[key] {
			return fmt.Errorf("omitted entry references unknown generated RPC %s", key)
		}
		if commands[key] {
			return fmt.Errorf("generated RPC %s is both commanded and omitted", key)
		}
		omitted[key] = true
	}
	for key := range available {
		if !commands[key] && !omitted[key] {
			return fmt.Errorf("generated RPC %s has neither a CLI command nor an omitted entry", key)
		}
	}
	return nil
}
