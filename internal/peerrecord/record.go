// Package peerrecord owns the cross-tier, same-user peer discovery file.
// It deliberately has no runtime-store dependency so Tier 1 and packaged
// desktop runtimes can share the exact codec and staleness rule. Records are
// keyed by canonical scenario instance slug: live uses the bare scenario name,
// while non-live variants use "scenario@variant".
package peerrecord

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/fsx"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const SchemaVersion = 1

// recordLocks serializes same-target operations inside one process. The file
// lock below provides the corresponding cross-process guarantee.
var recordLocks sync.Map // map[string]*sync.Mutex

// Feature is an explicit owner capability. Port names refer only to this record's Ports.
type Feature struct {
	OwnerVersion string `json:"owner_version"`
	Port         string `json:"port"`
}

type Record struct {
	Features      map[string]Feature `json:"features,omitempty"`
	SchemaVersion int                `json:"schema_version"`
	Scenario      string             `json:"scenario"`
	Instance      string             `json:"instance"`
	Tier          int                `json:"tier"`
	Version       string             `json:"version,omitempty"`
	Provides      []string           `json:"provides,omitempty"` // Published port capabilities, e.g. api or ui.
	OwnerPID      int                `json:"owner_pid"`
	Incarnation   string             `json:"incarnation"`
	StartedAt     time.Time          `json:"started_at"`
	Ports         map[string]int     `json:"ports"`
	AuthTokenPath string             `json:"auth_token_path"`
}

func Path(home, name string) string {
	if node, scenario, variant, err := cliutil.SplitAddress(name); err == nil && node == "" {
		name = cliutil.InstanceSlug(scenario, variant)
	}
	return filepath.Join(home, repocontractmeta.ProjectConfigDir, "peers", name+".json")
}

func Write(home string, record Record) error {
	if strings.TrimSpace(record.Scenario) == "" {
		return errors.New("peer record scenario is required")
	}
	target, scenario, variant, err := recordTarget(record.Scenario, record.Instance)
	if err != nil {
		return fmt.Errorf("invalid peer record instance: %w", err)
	}
	record.Scenario = scenario
	record.Instance = variant
	record.SchemaVersion = SchemaVersion
	if record.OwnerPID <= 0 {
		return errors.New("peer record owner PID is required")
	}
	// StartedAt is supervisor wall-clock metadata, not an OS process identity.
	// Persist the host-derived incarnation separately so PID reuse cannot make
	// an old peer record appear live.
	record.Incarnation, err = process.Incarnation(record.OwnerPID)
	if err != nil || record.Incarnation == "" {
		if err == nil {
			err = errors.New("empty process incarnation")
		}
		return fmt.Errorf("identify peer record owner process: %w", err)
	}
	release, err := acquireRecordLock(home, target)
	if err != nil {
		return err
	}
	defer release()
	if existing, readErr := readRecord(home, target); readErr == nil {
		if existing.OwnerPID <= 0 {
			return fmt.Errorf("refuse to replace peer record %s without an owner PID", target)
		}
		if platform.IsPIDRunning(existing.OwnerPID) && existing.OwnerPID != record.OwnerPID {
			return fmt.Errorf("peer record %s is owned by live process %d", target, existing.OwnerPID)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("refuse to replace invalid peer record %s: %w", target, readErr)
	}
	if record.Ports == nil {
		record.Ports = map[string]int{}
	}
	dir := filepath.Join(home, repocontractmeta.ProjectConfigDir, "peers")
	if err := os.MkdirAll(dir, tuning.PermPrivateDir); err != nil {
		return fmt.Errorf("create peer directory: %w", err)
	}
	if err := os.Chmod(dir, tuning.PermPrivateDir); err != nil {
		return fmt.Errorf("secure peer directory: %w", err)
	}
	payload, err := fsx.MarshalJSON(record)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(Path(home, target), payload, tuning.PermSecret)
}

func Read(home, name string) (Record, error) {
	record, err := readRecord(home, name)
	if err != nil {
		return Record{}, err
	}
	if record.OwnerPID <= 0 || !platform.IsPIDRunning(record.OwnerPID) || record.Incarnation == "" {
		return Record{}, os.ErrNotExist
	}
	incarnation, err := process.Incarnation(record.OwnerPID)
	if err != nil || incarnation != record.Incarnation {
		return Record{}, os.ErrNotExist
	}
	return record, nil
}

func readRecord(home, name string) (Record, error) {
	target, scenario, variant, err := targetAddress(name)
	if err != nil {
		return Record{}, fmt.Errorf("invalid peer record target: %w", err)
	}
	var record Record
	if err := fsx.ReadJSON(Path(home, target), &record); err != nil {
		return Record{}, err
	}
	recordTarget, recordScenario, recordVariant, recordErr := recordTarget(record.Scenario, record.Instance)
	if record.SchemaVersion != SchemaVersion || recordErr != nil || recordTarget != target || recordScenario != scenario || recordVariant != variant {
		return Record{}, fmt.Errorf("invalid peer record for %s", target)
	}
	return record, nil
}

// Remove deletes a peer record only when the current live owner is one of the
// caller's known owner processes. Dead-owner records are safe stale cleanup.
func Remove(home, name string, ownerPIDs ...int) error {
	target, _, _, err := targetAddress(name)
	if err != nil {
		return fmt.Errorf("invalid peer record target: %w", err)
	}
	release, err := acquireRecordLock(home, target)
	if err != nil {
		return err
	}
	defer release()
	record, err := readRecord(home, target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if record.OwnerPID <= 0 {
		return fmt.Errorf("refuse to remove peer record %s without an owner PID", target)
	}
	if platform.IsPIDRunning(record.OwnerPID) && !containsPID(ownerPIDs, record.OwnerPID) {
		return fmt.Errorf("peer record %s is owned by live process %d", target, record.OwnerPID)
	}
	err = os.Remove(Path(home, target))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func acquireRecordLock(home, target string) (func(), error) {
	dir := filepath.Join(home, repocontractmeta.ProjectConfigDir, "peers")
	if err := os.MkdirAll(dir, tuning.PermPrivateDir); err != nil {
		return nil, fmt.Errorf("create peer directory: %w", err)
	}
	if err := os.Chmod(dir, tuning.PermPrivateDir); err != nil {
		return nil, fmt.Errorf("secure peer directory: %w", err)
	}
	lockPath := Path(home, target) + ".lock"
	key := filepath.Clean(lockPath)
	value, _ := recordLocks.LoadOrStore(key, &sync.Mutex{})
	local := value.(*sync.Mutex)
	local.Lock()
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, tuning.PermLock)
	if err != nil {
		local.Unlock()
		return nil, fmt.Errorf("open peer record lock: %w", err)
	}
	if err := file.Chmod(tuning.PermLock); err != nil {
		_ = file.Close()
		local.Unlock()
		return nil, fmt.Errorf("secure peer record lock: %w", err)
	}
	unlock, err := platform.LockFile(file, false)
	if err != nil {
		_ = file.Close()
		local.Unlock()
		return nil, fmt.Errorf("lock peer record: %w", err)
	}
	return func() {
		unlock()
		_ = file.Close()
		local.Unlock()
	}, nil
}

func containsPID(pids []int, target int) bool {
	for _, pid := range pids {
		if pid == target {
			return true
		}
	}
	return false
}

func targetAddress(address string) (slug, scenario, variant string, err error) {
	node, scenario, variant, err := cliutil.SplitAddress(address)
	if err != nil {
		return "", "", "", err
	}
	if node != "" {
		return "", "", "", fmt.Errorf("peer record target must be local")
	}
	if cliutil.BareScenarioName(scenario) != scenario {
		return "", "", "", fmt.Errorf("peer record target has an invalid scenario name")
	}
	if strings.ContainsAny(scenario, `/\\`) || strings.ContainsAny(variant, `/\\`) {
		return "", "", "", fmt.Errorf("peer record target contains a path separator")
	}
	// Peer record identity is the supplied address/record identity, not the
	// caller's routing preference. ResolveVariant would let ambient shadow
	// configuration or a CLI override redirect a publisher's live record.
	if variant == "" {
		variant = cliutil.DefaultVariant
	}
	slug = cliutil.InstanceSlug(scenario, variant)
	_, variant = cliutil.SplitInstance(slug)
	if variant == "" {
		variant = cliutil.DefaultVariant
	}
	return slug, scenario, variant, nil
}

func recordTarget(scenario, instance string) (slug, normalizedScenario, normalizedInstance string, err error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" || cliutil.BareScenarioName(scenario) != scenario {
		return "", "", "", fmt.Errorf("record scenario must be a bare scenario name")
	}
	instance = strings.TrimSpace(instance)
	if strings.ContainsAny(scenario, `/\\`) || strings.ContainsAny(instance, `/\\`) {
		return "", "", "", fmt.Errorf("record identity contains a path separator")
	}
	address := scenario
	if instance != "" {
		address += "@" + instance
	}
	return targetAddress(address)
}
