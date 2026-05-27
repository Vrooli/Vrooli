package manifest

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

// baseExternalCLI returns a minimal valid external-cli manifest the durable_data
// tests layer a block onto, so each case isolates durable_data behavior.
func baseExternalCLI() ResourceManifest {
	return ResourceManifest{
		Name: "claude-code",
		CLI: &scenario.CLIConfig{
			Enabled: true,
			Command: "resource-claude-code",
			Adapter: scenario.CLIAdapterConfig{Kind: "go_module", ModuleDir: "cli"},
		},
		Driver:          "external-cli",
		Binary:          "claude",
		PortabilityTier: "partial",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "partial"},
	}
}

func boolPtr(b bool) *bool { return &b }

func TestValidateAcceptsDurableDataOnExternalCLI(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		Base: "$HOME/.claude",
		Entries: map[string]DurableDataEntry{
			"history":     {Path: "history.jsonl", Kind: "file", Regenerable: false, Rationale: "Prompt history."},
			"projects":    {Path: "projects", Kind: "dir", Regenerable: false},
			"state":       {Path: "state.sqlite", Kind: "file", Format: "sqlite", Regenerable: false},
			"credentials": {Path: ".credentials.json", Kind: "file", Sensitive: true, Regenerable: false},
		},
	}
	if err := Validate(m); err != nil {
		t.Fatalf("Validate() with valid durable_data: %v", err)
	}
}

func TestValidateRejectsDurableDataOnContainerDriver(t *testing.T) {
	m := baseExternalCLI()
	m.Driver = "docker-service"
	m.DurableData = &ResourceDurableData{
		Entries: map[string]DurableDataEntry{"x": {Path: "x", Kind: "dir"}},
	}
	err := Validate(m)
	if err == nil || !strings.Contains(err.Error(), "durable_data is only valid for host-filesystem drivers") {
		t.Fatalf("expected host-only driver rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataEmptyEntries(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{Base: "$HOME/.claude", Entries: map[string]DurableDataEntry{}}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "entries must not be empty") {
		t.Fatalf("expected empty-entries rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataBadKind(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		Entries: map[string]DurableDataEntry{"x": {Path: "x", Kind: "symlink"}},
	}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "kind must be") {
		t.Fatalf("expected bad-kind rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataTraversalPath(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		Entries: map[string]DurableDataEntry{"x": {Path: "../escape", Kind: "dir"}},
	}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "parent traversal") {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataAbsolutePath(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		Entries: map[string]DurableDataEntry{"x": {Path: "/etc/passwd", Kind: "file"}},
	}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "no leading slash") {
		t.Fatalf("expected absolute-path rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataBadBaseToken(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		Base:    "/var/lib/claude",
		Entries: map[string]DurableDataEntry{"x": {Path: "x", Kind: "dir"}},
	}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "must start with $HOME") {
		t.Fatalf("expected bad-base rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataHostOnlyFalse(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		HostOnly: boolPtr(false),
		Entries:  map[string]DurableDataEntry{"x": {Path: "x", Kind: "dir"}},
	}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "host_only must be true") {
		t.Fatalf("expected host_only rejection, got %v", err)
	}
}

func TestValidateRejectsDurableDataDuplicatePaths(t *testing.T) {
	m := baseExternalCLI()
	m.DurableData = &ResourceDurableData{
		Entries: map[string]DurableDataEntry{
			"a": {Path: "shared", Kind: "dir"},
			"b": {Path: "shared", Kind: "dir"},
		},
	}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "duplicates path") {
		t.Fatalf("expected duplicate-path rejection, got %v", err)
	}
}
