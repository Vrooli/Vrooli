// Package rootdecision owns the local locator for a pending Hub decision. It
// carries no authority: the Hub must authenticate and consume the answer.
package rootdecision

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/vrooli/api-core/rootrebaselinedecision"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/statecontract"
)

const pendingSchemaVersion = 1
const PendingFilename = "hub-decision.json"
const WatchdogFilename = "qualified-watchdog"

type pendingRecord struct {
	SchemaVersion int                          `json:"schema_version"`
	Terms         rootrebaselinedecision.Terms `json:"terms"`
}

func Load(path string) (rootrebaselinedecision.Terms, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return rootrebaselinedecision.Terms{}, nil
	}
	if err != nil {
		return rootrebaselinedecision.Terms{}, err
	}
	var record pendingRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return rootrebaselinedecision.Terms{}, err
	}
	if record.SchemaVersion != pendingSchemaVersion {
		return rootrebaselinedecision.Terms{}, os.ErrInvalid
	}
	return record.Terms, nil
}
func Save(path string, terms rootrebaselinedecision.Terms) error {
	data, err := json.Marshal(pendingRecord{SchemaVersion: pendingSchemaVersion, Terms: terms})
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(path, data, 0600)
}
func PersistentState() statecontract.Owner {
	return statecontract.Owner{Name: "root-rebaseline-hub-decision", WholeSource: true, Roots: []reflect.Type{reflect.TypeFor[pendingRecord]()}, Facts: map[string]any{"pendingSchemaVersion": pendingSchemaVersion, "path": "prepared-bundle/" + PendingFilename, "watchdog_path": "prepared-bundle/" + WatchdogFilename}}
}
