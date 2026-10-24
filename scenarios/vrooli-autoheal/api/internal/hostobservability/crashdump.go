package hostobservability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	EnvCrashDumpExportDir = "AUTOHEAL_CRASHDUMP_EXPORT_DIR"
	CrashDumpExportDir    = "/var/lib/vrooli/host-observability/crashdumps"
	CrashDumpSourceDir    = "/var/crash"
)

// CrashDump is one kdump-captured panic, as summarised by the kdump_observability
// collector. The raw vmcore never crosses this boundary — only the fields an
// incident report needs, plus the on-disk size so retention pressure is visible.
type CrashDump struct {
	// Stamp is kdump's own directory name, a YYYYMMDDHHMM timestamp. It is the
	// stable identity of the crash.
	Stamp string `json:"stamp"`
	// Summary is the exported dmesg tail's filename, relative to the export dir.
	Summary string `json:"summary"`
	// Reason is the panic banner — the "kernel BUG at …" or "Oops:" line.
	Reason string `json:"reason"`
	// Comm is the faulting task's command name, when the trace recorded one.
	Comm string `json:"comm"`
	// Bytes is the size of the retained crash directory, vmcore included.
	Bytes int64 `json:"bytes"`
}

// CrashDumpManifest is the collector's output.
type CrashDumpManifest struct {
	CollectedAt   string      `json:"collectedAt"`
	SourcePath    string      `json:"sourcePath"`
	RetainVmcores int         `json:"retainVmcores"`
	DumpCount     int         `json:"dumpCount"`
	Dumps         []CrashDump `json:"dumps"`
}

// CrashDumpExportPath resolves the export directory, honouring the environment
// override used by tests and non-standard layouts.
func CrashDumpExportPath() string {
	if override := strings.TrimSpace(os.Getenv(EnvCrashDumpExportDir)); override != "" {
		return override
	}
	return CrashDumpExportDir
}

// ReadCrashDumpManifest loads the collector manifest from an export directory.
// A missing manifest is reported as-is rather than as an empty result: "no
// collector has run here" and "the host has not crashed" are different facts and
// must not collapse into the same answer.
func ReadCrashDumpManifest(exportDir string) (CrashDumpManifest, error) {
	raw, err := os.ReadFile(filepath.Join(exportDir, ManifestFilename))
	if err != nil {
		return CrashDumpManifest{}, err
	}
	var manifest CrashDumpManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return CrashDumpManifest{}, fmt.Errorf("parse crash dump manifest: %w", err)
	}
	// Newest first: an incident reporter cares about the most recent crash, and
	// kdump's stamp format sorts lexicographically in timestamp order.
	sort.SliceStable(manifest.Dumps, func(i, j int) bool {
		return manifest.Dumps[i].Stamp > manifest.Dumps[j].Stamp
	})
	return manifest, nil
}

// Newest returns the most recent crash dump, if any.
func (m CrashDumpManifest) Newest() (CrashDump, bool) {
	if len(m.Dumps) == 0 {
		return CrashDump{}, false
	}
	return m.Dumps[0], true
}

// TotalBytes is the disk currently held by retained vmcores. Each is roughly the
// size of system RAM, so this is worth reporting even when nothing is wrong.
func (m CrashDumpManifest) TotalBytes() int64 {
	var total int64
	for _, dump := range m.Dumps {
		total += dump.Bytes
	}
	return total
}
