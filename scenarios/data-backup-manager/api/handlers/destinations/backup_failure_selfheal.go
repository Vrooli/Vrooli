package destinations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/repohealth"
)

// AutoAcceptSource marks an exception recorded by policy rather than by a
// verified human decision. Exception notices skip these so an accepted,
// harmless leftover does not become a daily nag.
const AutoAcceptSource = "policy:leftover-temp"

// autoAcceptLeftoverTempsEnabled is the standing policy for the only
// non-destructive exception class: unreadable leftover *session-marker temp
// files*, which are unfinished uploads, not saved backup data.
//
// It is on by default because the alternative is an operator answering the same
// question every time a drive is unplugged mid-write. Set
// DBM_AUTO_ACCEPT_LEFTOVER_TEMP=0 to require an explicit decision for every
// leftover.
func autoAcceptLeftoverTempsEnabled() bool {
	// Tests must opt in explicitly so a harness that is exercising the decision
	// flow still sees the question it is asserting on.
	defaultOn := !testing.Testing()
	value, ok := os.LookupEnv("DBM_AUTO_ACCEPT_LEFTOVER_TEMP")
	if !ok {
		return defaultOn
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// autoAcceptLeftoverTemps records a policy exception for every blocking entry
// when, and only when, all of them are EIO-unreadable leftover session-marker
// temp files. It never touches the drive: the only write is a state file in the
// DBM state root.
//
// Snapshot data, manifest data, and anything ambiguous are refused, so they
// still reach the verified decision flow.
func (w *BackupFailureWorker) autoAcceptLeftoverTemps(ctx context.Context, dest destinations.Destination, diag repohealth.Diagnosis, planIDs []string) (bool, error) {
	if !autoAcceptLeftoverTempsEnabled() || w.exceptions == nil || w.usage == nil || w.usage.Scanner == nil {
		return false, nil
	}
	if diag.Cause != repohealth.CauseLeftoverTempUnreadable || len(diag.Unreadable) == 0 {
		return false, nil
	}
	fsys := w.usage.Scanner.Filesystem()
	now := w.now().UTC()
	for _, entry := range diag.Unreadable {
		if entry.Class != repohealth.ClassLeftoverTemp || entry.Problem != repohealth.ProblemIO {
			return false, nil
		}
		info, err := fsys.Lstat(filepath.Join(diag.RepositoryRoot, filepath.FromSlash(entry.RelPath)))
		switch {
		case err != nil && repohealth.ProblemFor(err) == entry.Problem:
			// still unreadable with the same kind of failure
		case err == nil && info.Mode().IsRegular():
			// readable again; nothing to except
		default:
			return false, nil
		}
	}
	recorded := 0
	for _, entry := range diag.Unreadable {
		ex := repohealth.Exception{
			DestinationID:    dest.ID,
			RelPath:          entry.RelPath,
			Class:            entry.Class,
			BlobKind:         entry.BlobKind,
			Problem:          entry.Problem,
			ApprovedBy:       "dbm-self-heal",
			ApprovedBySource: AutoAcceptSource,
			ApprovedAt:       now,
		}
		if err := w.exceptions.Add(ctx, ex); err != nil {
			return false, err
		}
		recorded++
	}
	if recorded == 0 {
		return false, nil
	}
	w.usage.InvalidateScan(diag.RepositoryRoot)
	w.logger.Printf("self-heal: recorded %d unreadable leftover temp file(s) on destination %s; no drive data was touched", recorded, dest.ID)
	w.retry(ctx, planIDs)
	return true, nil
}
