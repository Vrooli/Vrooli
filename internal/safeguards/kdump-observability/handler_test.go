package kdumpobservability

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func testManifest() hostreqkit.SafeguardManifest {
	return hostreqkit.SafeguardManifest{Name: "kdump_observability"}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", SupportsSystemd: true}
}

func req() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "kdump_observability", Kind: hostreqspec.KindSafeguard, Required: true}
}

// The handler fallback and the manifest schema must agree on what an
// unconfigured requirement means.
func TestDefaultsMatchManifest(t *testing.T) {
	raw, err := os.ReadFile("safeguard.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Config struct {
			Properties struct {
				RetainVmcores struct {
					Default float64 `json:"default"`
				} `json:"retain_vmcores"`
			} `json:"properties"`
		} `json:"config"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if int(manifest.Config.Properties.RetainVmcores.Default) != defaultRetainVmcores {
		t.Fatalf("manifest default %v, handler default %d",
			manifest.Config.Properties.RetainVmcores.Default, defaultRetainVmcores)
	}
}

// Retaining zero dumps would delete the evidence for the crash that just
// happened, so a nonsensical declared value falls back rather than being obeyed.
func TestRetainVmcoresRejectsUnsafeValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		want   int
	}{
		{"nil config", nil, defaultRetainVmcores},
		{"absent key", map[string]any{}, defaultRetainVmcores},
		{"zero", map[string]any{"retain_vmcores": float64(0)}, defaultRetainVmcores},
		{"negative", map[string]any{"retain_vmcores": float64(-3)}, defaultRetainVmcores},
		{"wrong type", map[string]any{"retain_vmcores": "many"}, defaultRetainVmcores},
		{"json number", map[string]any{"retain_vmcores": float64(5)}, 5},
		{"int", map[string]any{"retain_vmcores": 4}, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := retainVmcores(tc.config); got != tc.want {
				t.Fatalf("retainVmcores = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	status := NewHandler(testManifest()).Inspect(hostreqkit.Host{OS: "darwin"}, req())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q, want unsupported", status.SupportClass)
	}
}

func TestInspectRequiresSystemd(t *testing.T) {
	host := linuxHost()
	host.SupportsSystemd = false
	status := NewHandler(testManifest()).Inspect(host, req())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q, want unsupported without systemd", status.SupportClass)
	}
}

// The export must never copy a vmcore. A vmcore is roughly the size of system
// RAM, and the whole point of the summary is that it is small enough to keep.
func TestCollectorNeverCopiesTheVmcore(t *testing.T) {
	script := collectorContent(defaultRetainVmcores)
	if strings.Contains(script, "vmcore") && strings.Contains(script, "cp -- \"$dir/vmcore\"") {
		t.Fatal("collector must not copy vmcore files into the export directory")
	}
	if !strings.Contains(script, "tail -n") {
		t.Fatal("collector should export a bounded tail of the crash dmesg")
	}
}

// Pruning is the one destructive thing this safeguard does, so its guards are
// worth pinning: only numeric kdump directories, only under /var/crash, and
// only once a summary exists.
func TestCollectorPruningIsGuarded(t *testing.T) {
	script := collectorContent(defaultRetainVmcores)
	for _, guard := range []string{
		`grep -E '^[0-9]+$'`,             // only kdump's own timestamp directories
		`[ -f "$dst/$old.dmesg" ]`,       // only when the summary was exported
		`[ -d "$src/$old" ] || continue`, // only real directories
		`rm -rf -- "$src/$old"`,          // scoped to the crash directory
	} {
		if !strings.Contains(script, guard) {
			t.Errorf("pruning guard missing: %s", guard)
		}
	}
}

func TestCollectorHonoursRetentionSetting(t *testing.T) {
	if !strings.Contains(collectorContent(7), "retain=7") {
		t.Fatal("collector should embed the resolved retention count")
	}
}

// The exported artifacts must be group-readable, not world-readable: the whole
// point is a narrow channel to the observability group.
func TestExportedArtifactsAreGroupScoped(t *testing.T) {
	script := collectorContent(defaultRetainVmcores)
	if !strings.Contains(script, `chmod 0640`) {
		t.Error("exported files should be 0640")
	}
	if !strings.Contains(script, `install -d -o root -g "$group" -m 0750 "$dst"`) {
		t.Error("export directory should be 0750 root:vrooli-observability")
	}
}

// A summary is only useful to an incident reporter if it carries the fields the
// report leads with.
func TestManifestCarriesPanicSummaryFields(t *testing.T) {
	script := collectorContent(defaultRetainVmcores)
	for _, field := range []string{`\"stamp\":`, `\"reason\":`, `\"comm\":`, `\"bytes\":`} {
		if !strings.Contains(script, field) {
			t.Errorf("manifest should carry %s", field)
		}
	}
	if !strings.Contains(script, "kernel BUG at") {
		t.Error("summary extraction should recognise a kernel BUG banner")
	}
}

func TestUnitsReferenceTheCollector(t *testing.T) {
	if !strings.Contains(serviceContent(), collectorPath) {
		t.Error("service should run the installed collector")
	}
	if !strings.Contains(serviceContent(), "ConditionPathExists="+crashSourceDir) {
		t.Error("service should be conditional on the crash directory existing")
	}
	if !strings.Contains(timerContent(), "Persistent=true") {
		t.Error("timer should be persistent so a missed boot still collects")
	}
}
