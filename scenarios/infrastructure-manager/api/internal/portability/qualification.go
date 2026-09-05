package portability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const qualificationObservationPath = ".vrooli/portability/qualification-observations.json"

// RecordQualificationObservation appends/replaces the latest observation for
// one resource/platform cell. Callers must supply the node's reported OS and
// architecture and an actual health result; declaration-only callers cannot
// manufacture qualification.
func RecordQualificationObservation(root string, observation QualificationObservation) error {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(observation.Resource) == "" || strings.TrimSpace(observation.Node) == "" || strings.TrimSpace(observation.RunID) == "" {
		return fmt.Errorf("qualification observation requires root, resource, node, and run_id")
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = time.Now().UTC()
	}
	path := filepath.Join(root, qualificationObservationPath)
	observations := readQualificationObservations(path)
	key := observation.Resource + ":" + observation.HostOS + ":" + observation.Architecture
	updated := false
	for i := range observations {
		if observations[i].Resource+":"+observations[i].HostOS+":"+observations[i].Architecture == key {
			if observation.ObservedAt.After(observations[i].ObservedAt) {
				observations[i] = observation
			}
			updated = true
			break
		}
	}
	if !updated {
		observations = append(observations, observation)
	}
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].Resource+observations[i].HostOS+observations[i].Architecture < observations[j].Resource+observations[j].HostOS+observations[j].Architecture
	})
	data, err := json.MarshalIndent(observations, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func readQualificationObservations(path string) []QualificationObservation {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var observations []QualificationObservation
	if json.Unmarshal(data, &observations) != nil {
		return nil
	}
	return observations
}

func (r *Reader) qualificationObservations() []QualificationObservation {
	return readQualificationObservations(filepath.Join(r.root, qualificationObservationPath))
}
