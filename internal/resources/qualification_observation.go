package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const qualificationObservationRelativePath = ".vrooli/portability/qualification-observations.json"

type resourceQualificationObservation struct {
	Resource     string    `json:"resource"`
	HostOS       string    `json:"host_os"`
	Architecture string    `json:"architecture"`
	Node         string    `json:"node"`
	RunID        string    `json:"run_id"`
	ObservedAt   time.Time `json:"observed_at"`
	HealthPassed bool      `json:"health_passed"`
}

var qualificationObservationMu sync.Mutex

// recordQualificationObservation records a successful, locally observed
// resource start. Portability consumes this evidence to promote a declared
// platform cell; a failed write must not make an otherwise healthy resource
// fail to start.
func recordQualificationObservation(root, resource string) error {
	root = strings.TrimSpace(root)
	resource = strings.TrimSpace(resource)
	if root == "" || resource == "" {
		return fmt.Errorf("qualification observation requires root and resource")
	}
	node, err := os.Hostname()
	if err != nil || strings.TrimSpace(node) == "" {
		return fmt.Errorf("resolve qualification node: %w", err)
	}
	now := time.Now().UTC()
	observation := resourceQualificationObservation{
		Resource:     resource,
		HostOS:       runtime.GOOS,
		Architecture: runtime.GOARCH,
		Node:         node,
		RunID:        fmt.Sprintf("resource-start-%d", now.UnixNano()),
		ObservedAt:   now,
		HealthPassed: true,
	}

	qualificationObservationMu.Lock()
	defer qualificationObservationMu.Unlock()

	path := filepath.Join(root, qualificationObservationRelativePath)
	observations := make([]resourceQualificationObservation, 0)
	if data, readErr := os.ReadFile(path); readErr == nil {
		if err := json.Unmarshal(data, &observations); err != nil {
			return fmt.Errorf("decode qualification observations: %w", err)
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("read qualification observations: %w", readErr)
	}
	key := observation.Resource + ":" + observation.HostOS + ":" + observation.Architecture
	replaced := false
	for i := range observations {
		if observations[i].Resource+":"+observations[i].HostOS+":"+observations[i].Architecture == key {
			if observation.ObservedAt.After(observations[i].ObservedAt) {
				observations[i] = observation
			}
			replaced = true
			break
		}
	}
	if !replaced {
		observations = append(observations, observation)
	}
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].Resource+observations[i].HostOS+observations[i].Architecture < observations[j].Resource+observations[j].HostOS+observations[j].Architecture
	})
	data, err := json.MarshalIndent(observations, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create qualification observation directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".qualification-observations-*.tmp")
	if err != nil {
		return fmt.Errorf("create qualification observation temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write qualification observations: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync qualification observations: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("install qualification observations: %w", err)
	}
	return nil
}
