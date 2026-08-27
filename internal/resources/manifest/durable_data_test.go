package manifest

import (
	"strings"
	"testing"
)

// baseExternalCLI returns a minimal valid external-cli manifest the durable_data
// tests layer a block onto, so each case isolates durable_data behavior.
func baseExternalCLI() ResourceManifest {
	return ResourceManifest{
		Name:      "claude-code",
		CLI:       validCLI("resource-claude-code"),
		Driver:    "external-cli",
		Binary:    "claude",
		Platforms: ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "partial"},
	}
}

func boolPtr(b bool) *bool { return &b }

func assertDurableDataRejected(t *testing.T, mutate func(*ResourceManifest), want string) {
	t.Helper()
	manifest := baseExternalCLI()
	mutate(&manifest)
	err := Validate(manifest)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q rejection, got %v", want, err)
	}
}

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
	assertDurableDataRejected(t, func(m *ResourceManifest) {
		m.Driver = "docker-service"
		m.DurableData = &ResourceDurableData{
			Entries: map[string]DurableDataEntry{"x": {Path: "x", Kind: "dir"}},
		}
	}, "durable_data is only valid for host-filesystem drivers")
}

func TestValidateRejectsDurableDataEmptyEntries(t *testing.T) {
	assertDurableDataRejected(t, func(m *ResourceManifest) {
		m.DurableData = &ResourceDurableData{Base: "$HOME/.claude", Entries: map[string]DurableDataEntry{}}
	}, "entries must not be empty")
}

func TestValidateRejectsDurableDataBadKind(t *testing.T) {
	assertDurableDataRejected(t, func(m *ResourceManifest) {
		m.DurableData = &ResourceDurableData{
			Entries: map[string]DurableDataEntry{"x": {Path: "x", Kind: "symlink"}},
		}
	}, "kind must be")
}

func TestValidateRejectsDurableDataTraversalPath(t *testing.T) {
	assertDurableDataRejected(t, func(m *ResourceManifest) {
		m.DurableData = &ResourceDurableData{
			Entries: map[string]DurableDataEntry{"x": {Path: "../escape", Kind: "dir"}},
		}
	}, "parent traversal")
}

func TestValidateRejectsDurableDataAbsolutePath(t *testing.T) {
	assertDurableDataRejected(t, func(m *ResourceManifest) {
		m.DurableData = &ResourceDurableData{
			Entries: map[string]DurableDataEntry{"x": {Path: "/etc/passwd", Kind: "file"}},
		}
	}, "no leading slash")
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
