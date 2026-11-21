package backup

import "scenario-to-cloud/domain"

// PredecessorState is what the cloud side observed about the data a
// deployment already holds before a new revision touches it. It is derived
// from the actual predecessor (inventory counts, the recorded schema
// version), never from what the plan hopes.
type PredecessorState struct {
	// Exists is false for a first deployment: no predecessor release holds
	// any binding on the target.
	Exists bool `json:"exists"`
	// HasData is true when any declared binding holds rows or objects.
	HasData bool `json:"has_data"`
	// SchemaVersion is the schema the predecessor's data is at; empty when
	// the workload never versioned its schema.
	SchemaVersion string `json:"schema_version"`
	// Versioned is true when the workload declares versioned migrations
	// (storage-steer tier 3: production evolution).
	Versioned bool `json:"versioned"`
}

// SelectPosture chooses the storage-steer migration posture from the actual
// predecessor state:
//
//	greenfield            no predecessor data: declarative schema creation
//	greenfield_with_data  data exists but the schema is unversioned: a
//	                      one-shot transformation of preserved data
//	production_evolution  data exists under a versioned schema: the complete
//	                      ordered delta must be tested against predecessor data
//
// The posture is recorded on every recovery point so the restore side knows
// which schema story the captured data carries.
func SelectPosture(state PredecessorState) string {
	switch {
	case !state.Exists || !state.HasData:
		return domain.MigrationPostureGreenfield
	case state.Versioned || state.SchemaVersion != "":
		return domain.MigrationPostureProductionEvolution
	default:
		return domain.MigrationPostureGreenfieldWithData
	}
}
