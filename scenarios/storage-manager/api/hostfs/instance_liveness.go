package hostfs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstanceRegistryLiveness answers "is this scenario instance running?" from
// the control plane's own process registry at
// <home>/.vrooli/processes/scenarios/<slug>/, where <slug> is the canonical
// "<scenario>@<variant>" instance key.
//
// The registry is the authority because it is what `vrooli scenario start`
// writes and what `vrooli scenario stop` clears. A record alone is not proof of
// life -- a killed process leaves its record behind -- so the recorded pid is
// probed as well. Both must agree before an instance counts as running.
type InstanceRegistryLiveness struct {
	root string
}

// NewInstanceRegistryLiveness builds a liveness reader over the process
// registry's scenarios directory.
func NewInstanceRegistryLiveness(scenarioRecordsRoot string) *InstanceRegistryLiveness {
	return &InstanceRegistryLiveness{root: filepath.Clean(strings.TrimSpace(scenarioRecordsRoot))}
}

// processRecord is the subset of a lifecycle step record this reader needs.
type processRecord struct {
	PID    int    `json:"pid"`
	Status string `json:"status"`
}

// RunningInstances returns every instance slug with at least one live recorded
// process.
//
// An unreadable registry root is an error, not an empty set. Callers use this
// to decide whether storage may be reclaimed, and "I could not look" must never
// be indistinguishable from "nothing is running".
func (l *InstanceRegistryLiveness) RunningInstances(ctx context.Context) (map[string]struct{}, error) {
	if l.root == "" || l.root == "." {
		return nil, fmt.Errorf("instance process registry root is unset")
	}
	entries, err := os.ReadDir(l.root)
	if err != nil {
		if os.IsNotExist(err) {
			// No registry directory at all means the lifecycle has never
			// started anything on this host. That is a real, readable answer.
			return map[string]struct{}{}, nil
		}
		return nil, fmt.Errorf("read instance process registry: %w", err)
	}
	running := make(map[string]struct{})
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		alive, err := l.instanceAlive(ctx, filepath.Join(l.root, slug))
		if err != nil {
			return nil, err
		}
		if alive {
			running[slug] = struct{}{}
		}
	}
	return running, nil
}

func (l *InstanceRegistryLiveness) instanceAlive(ctx context.Context, dir string) (bool, error) {
	records, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read instance records: %w", err)
	}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if record.IsDir() || !strings.HasSuffix(record.Name(), ".json") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(dir, record.Name()))
		if err != nil {
			// One unreadable record must not be read as "stopped". Surface it
			// so the caller fails closed.
			return false, fmt.Errorf("read instance record %s: %w", record.Name(), err)
		}
		var parsed processRecord
		if json.Unmarshal(contents, &parsed) != nil {
			// A malformed record carries no pid to probe. It is also not
			// evidence of a stopped instance, so treat the instance as alive.
			return true, nil
		}
		if parsed.PID <= 0 {
			continue
		}
		if parsed.Status != "" && parsed.Status != "running" {
			continue
		}
		if pidAlive(parsed.PID) {
			return true, nil
		}
	}
	return false, nil
}
