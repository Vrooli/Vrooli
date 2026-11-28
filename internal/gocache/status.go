package gocache

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Activation reports whether new go commands on this host use the protocol:
// the user's go env file names the installed helper as GOCACHEPROG.
type Activation struct {
	Active            bool   `json:"active"`
	Reason            string `json:"reason,omitempty"`
	Program           string `json:"program"`
	ProgramInstalled  bool   `json:"program_installed"`
	EnvFile           string `json:"env_file,omitempty"`
	ConfiguredProgram string `json:"configured_program,omitempty"`
}

// ReadActivation inspects this user's go env file. The file, not this
// process's environment, is what every new go command on the host reads.
func ReadActivation(program string) Activation {
	activation := Activation{Program: program}
	if !protocolSupported || !auditSupported {
		activation.Reason = "the protocol requires Linux"
		return activation
	}
	if info, err := os.Stat(program); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
		activation.ProgramInstalled = true
	}
	settings := resolveGoSettings(func(key string) string {
		// GOCACHEPROG and GOCACHE in this process's environment do not reach
		// other go commands; only the env file location does.
		if key == "GOCACHEPROG" {
			return ""
		}
		return os.Getenv(key)
	}, ownDirs())
	activation.EnvFile = settings.EnvFile
	activation.ConfiguredProgram = settings.Program
	switch {
	case !filepath.IsAbs(program):
		activation.Reason = "helper path is not absolute"
	case settings.EnvFile == "":
		activation.Reason = "no go env file resolves for this user"
	case settings.Program == "":
		activation.Reason = "GOCACHEPROG is not set in " + settings.EnvFile
	case settings.Program != program:
		activation.Reason = "GOCACHEPROG in " + settings.EnvFile + " is " + settings.Program
	case !activation.ProgramInstalled:
		activation.Reason = "the configured helper " + program + " is not installed; every go command will fail until it is reinstalled or the protocol is deactivated"
	default:
		activation.Active = true
	}
	return activation
}

// Status is a cheap read of a store's protocol state; it does not scan the
// store.
type Status struct {
	Schema       string       `json:"schema"`
	Cache        string       `json:"cache"`
	StoreID      string       `json:"store_id,omitempty"`
	Activation   Activation   `json:"activation"`
	LiveSessions int          `json:"live_sessions"`
	Audit        *AuditReport `json:"audit,omitempty"`
	AuditError   string       `json:"audit_error,omitempty"`
	LastReclaim  *Receipt     `json:"last_reclaim,omitempty"`
	ObservedAt   time.Time    `json:"observed_at"`
}

// ReadStatus reports activation, live sessions, the go-command audit, and
// the last reclaim receipt for a store.
func ReadStatus(store Store, program string, now func() time.Time) (Status, error) {
	now = clock(now)
	status := Status{Schema: StatusSchema, Cache: store.Root, Activation: ReadActivation(program), ObservedAt: now().UTC()}
	state, err := openState(store)
	if err != nil {
		return status, err
	}
	defer state.Close()
	if status.StoreID, err = state.StoreID(); err != nil {
		return status, err
	}
	status.LiveSessions, _ = countLiveSessions(state.sessionsDir())
	reason, report := auditStore(store, program, now)
	status.Audit = report
	if reason != "" && (report == nil || report.Clear()) {
		status.AuditError = reason
	}
	var receipt Receipt
	if err := readJSON(state.path(receiptFileName), &receipt); err == nil {
		status.LastReclaim = &receipt
	}
	return status, nil
}

// countLiveSessions counts journals whose session still holds its lock. It
// never removes anything.
func countLiveSessions(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	live := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), pinJournalSuffix) {
			continue
		}
		file, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		acquired, err := tryLock(file, false)
		switch {
		case err == nil && !acquired:
			live++
		case acquired:
			_ = unlock(file)
		}
		file.Close()
	}
	return live, nil
}
