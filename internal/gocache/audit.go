package gocache

import (
	"debug/buildinfo"
	"time"
)

// Main package paths that identify processes by their embedded build
// information rather than by name.
const (
	goCommandMainPath = "cmd/go"
	ProgramMainPath   = "github.com/vrooli/vrooli/cmd/vrooli-gocacheprog"
	// ProgramName is the installed helper's file name.
	ProgramName = "vrooli-gocacheprog"
)

// ProcessFinding names a live go command reclaim cannot prove safe.
type ProcessFinding struct {
	PID     int    `json:"pid"`
	Reason  string `json:"reason"`
	Command string `json:"command,omitempty"`
}

// AuditReport classifies every live go command against one store.
type AuditReport struct {
	ObservedAt time.Time `json:"observed_at"`
	// GoCommands counts live cmd/go processes seen on the host.
	GoCommands int `json:"go_commands"`
	// Mediated counts go commands proven to reach this store only through a
	// session: they have a live helper child, or they will start one on first
	// cache use.
	Mediated int `json:"mediated"`
	// OtherStores counts go commands provably using a different GOCACHE.
	OtherStores int              `json:"other_stores"`
	Unmediated  []ProcessFinding `json:"unmediated,omitempty"`
}

// Clear reports whether no go command could be using the store outside the
// protocol.
func (r AuditReport) Clear() bool { return len(r.Unmediated) == 0 }

// storeAuditor classifies live go commands against one store.
type storeAuditor interface {
	run() (AuditReport, error)
}

// newStoreAuditor is the audit seam; the platform auditor is the only
// production implementation.
var newStoreAuditor = func(store Store, program string, now func() time.Time) (storeAuditor, error) {
	return newAuditor(store, program, now)
}

// executable is the identity a process's embedded build information gives.
type executable struct {
	mainPath  string
	goVersion string
}

func readExecutable(path string) (executable, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return executable{}, err
	}
	return executable{mainPath: info.Path, goVersion: info.GoVersion}, nil
}
