package certification

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed dm_lane_owner_routes.json
var dmLaneOwnerRouteBytes embed.FS

var dmlaneOwnerScenarioPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var dmlaneWorkstreamIDPattern = regexp.MustCompile(`^CD[0-9]+$`)

// DMLaneGap is an actionable owner-routed projection of one non-passing lane
// stage. Identity includes the exact reason and evidence, so identical
// observations deduplicate while a changed gap remains a separate record.
type DMLaneGap struct {
	SchemaVersion   int         `json:"schema_version"`
	ID              string      `json:"id"`
	Scenario        string      `json:"scenario"`
	ScenarioOwner   string      `json:"scenario_owner"`
	WorkstreamOwner string      `json:"workstream_owner"`
	WorkstreamID    string      `json:"workstream_id"`
	WorkstreamRef   string      `json:"workstream_ref"`
	ProfileID       string      `json:"profile_id"`
	OperationID     string      `json:"operation_id,omitempty"`
	Stage           string      `json:"stage"`
	Status          DMLaneState `json:"status"`
	Reason          string      `json:"reason"`
	EvidenceRef     string      `json:"evidence_ref,omitempty"`
}

type dmlaneWorkstream struct {
	ID            string `json:"id"`
	WorkstreamRef string `json:"workstream_ref"`
	OwnerScenario string `json:"owner_scenario"`
}

type dmlaneOwnerRoutes struct {
	SchemaVersion int                `json:"schema_version"`
	Source        string             `json:"source"`
	Workstreams   []dmlaneWorkstream `json:"workstreams"`
	StageRoutes   map[string]string  `json:"stage_routes"`
}

func loadDMLaneOwnerRoutes() (dmlaneOwnerRoutes, error) {
	raw, err := dmLaneOwnerRouteBytes.ReadFile("dm_lane_owner_routes.json")
	if err != nil {
		return dmlaneOwnerRoutes{}, fmt.Errorf("read DM lane owner routes: %w", err)
	}
	var routes dmlaneOwnerRoutes
	if err := json.Unmarshal(raw, &routes); err != nil {
		return dmlaneOwnerRoutes{}, fmt.Errorf("decode DM lane owner routes: %w", err)
	}
	if err := validateDMLaneOwnerRoutes(routes); err != nil {
		return dmlaneOwnerRoutes{}, err
	}
	return routes, nil
}

func validateDMLaneOwnerRoutes(routes dmlaneOwnerRoutes) error {
	if routes.SchemaVersion != 1 || routes.Source != "research/topics/compute-and-deployment/MISSION.md" || len(routes.Workstreams) == 0 {
		return fmt.Errorf("DM lane owner route registry is missing version, source, or workstreams")
	}
	workstreams := make(map[string]dmlaneWorkstream, len(routes.Workstreams))
	for _, workstream := range routes.Workstreams {
		if !dmlaneWorkstreamIDPattern.MatchString(workstream.ID) || workstream.WorkstreamRef != "compute-and-deployment#"+workstream.ID || !dmlaneOwnerScenarioPattern.MatchString(workstream.OwnerScenario) {
			return fmt.Errorf("DM lane owner route registry has an invalid workstream owner")
		}
		if _, exists := workstreams[workstream.ID]; exists {
			return fmt.Errorf("DM lane owner route registry has duplicate workstream %q", workstream.ID)
		}
		workstreams[workstream.ID] = workstream
	}
	for _, stage := range []string{"acquire", "deploy", "verify", "teardown", "lease-renewal"} {
		workstreamID, ok := routes.StageRoutes[stage]
		if !ok {
			return fmt.Errorf("DM lane owner route registry has no owner for stage %q", stage)
		}
		if _, ok := workstreams[workstreamID]; !ok {
			return fmt.Errorf("DM lane stage %q names unknown workstream %q", stage, workstreamID)
		}
	}
	for stage := range routes.StageRoutes {
		switch stage {
		case "acquire", "deploy", "verify", "teardown", "lease-renewal":
		default:
			return fmt.Errorf("DM lane owner route registry names unsupported stage %q", stage)
		}
	}
	return nil
}

func resolveDMLaneWorkstreamOwner(stage string, routes dmlaneOwnerRoutes) (dmlaneWorkstream, error) {
	workstreamID, ok := routes.StageRoutes[stage]
	if !ok {
		return dmlaneWorkstream{}, fmt.Errorf("no responsible workstream owner registered for DM lane stage %q", stage)
	}
	for _, workstream := range routes.Workstreams {
		if workstream.ID == workstreamID {
			return workstream, nil
		}
	}
	return dmlaneWorkstream{}, fmt.Errorf("DM lane stage %q resolves to unknown workstream %q", stage, workstreamID)
}

// ProjectDMLaneGaps emits a routing record for every stage that is not passed.
// A scenario owner is derived from the generated DM profile identity; the
// workstream owner is explicit and unknown stages fail closed.
func ProjectDMLaneGaps(verdicts []DMLaneVerdict) ([]DMLaneGap, error) {
	routes, err := loadDMLaneOwnerRoutes()
	if err != nil {
		return nil, err
	}
	var gaps []DMLaneGap
	for _, verdict := range verdicts {
		if !dmlaneOwnerScenarioPattern.MatchString(verdict.Scenario) || strings.TrimSpace(verdict.ProfileID) == "" {
			return nil, fmt.Errorf("cannot route DM lane gap without scenario/profile identity")
		}
		for _, stage := range verdict.Stages {
			if stage.Status == DMLanePassed {
				continue
			}
			workstream, err := resolveDMLaneWorkstreamOwner(stage.Stage, routes)
			if err != nil {
				return nil, err
			}
			gap := DMLaneGap{SchemaVersion: 1, Scenario: verdict.Scenario, ScenarioOwner: "scenario:" + verdict.Scenario,
				WorkstreamOwner: workstream.OwnerScenario, WorkstreamID: workstream.ID, WorkstreamRef: workstream.WorkstreamRef,
				ProfileID: verdict.ProfileID, OperationID: verdict.OperationID,
				Stage: stage.Stage, Status: stage.Status, Reason: stage.Reason, EvidenceRef: stage.EvidenceRef}
			gap.ID, err = dmlaneGapID(gap)
			if err != nil {
				return nil, err
			}
			gaps = append(gaps, gap)
		}
	}
	return gaps, nil
}

func dmlaneGapID(gap DMLaneGap) (string, error) {
	gap.ID = ""
	raw, err := json.Marshal(gap)
	if err != nil {
		return "", fmt.Errorf("encode DM lane gap identity: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// ReadDMLaneGaps loads only persisted, identity-verified gap records. The
// Swarm Manager handoff consumes these files rather than reconstructing gaps
// from an in-memory projection, preserving the durable evidence boundary.
func ReadDMLaneGaps(paths []string) ([]DMLaneGap, error) {
	routes, err := loadDMLaneOwnerRoutes()
	if err != nil {
		return nil, err
	}
	result := make([]DMLaneGap, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read persisted DM lane gap %q: %w", path, err)
		}
		var gap DMLaneGap
		if err := json.Unmarshal(raw, &gap); err != nil {
			return nil, fmt.Errorf("decode persisted DM lane gap %q: %w", path, err)
		}
		if gap.SchemaVersion != 1 || (gap.Status != DMLaneBlocked && gap.Status != DMLaneFailed && gap.Status != DMLaneNotRun) || !dmlaneOwnerScenarioPattern.MatchString(gap.Scenario) || gap.ScenarioOwner != "scenario:"+gap.Scenario || strings.TrimSpace(gap.ProfileID) == "" || strings.TrimSpace(gap.Reason) == "" {
			return nil, fmt.Errorf("persisted DM lane gap %q is incomplete or not actionable", path)
		}
		workstream, err := resolveDMLaneWorkstreamOwner(gap.Stage, routes)
		if err != nil || gap.WorkstreamID != workstream.ID || gap.WorkstreamOwner != workstream.OwnerScenario || gap.WorkstreamRef != workstream.WorkstreamRef {
			return nil, fmt.Errorf("persisted DM lane gap %q has an unknown or mismatched owner route", path)
		}
		id, err := dmlaneGapID(gap)
		if err != nil || gap.ID != id {
			return nil, fmt.Errorf("persisted DM lane gap %q has an invalid content identity", path)
		}
		if seen[gap.ID] {
			continue
		}
		seen[gap.ID] = true
		result = append(result, gap)
	}
	return result, nil
}

// WriteDMLaneGaps stores immutable, content-addressed routing records. A
// repeated write is idempotent when the existing bytes match the record.
func WriteDMLaneGaps(dir string, verdicts []DMLaneVerdict) ([]string, error) {
	gaps, err := ProjectDMLaneGaps(verdicts)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create DM lane gap directory: %w", err)
	}
	paths := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		raw, err := json.MarshalIndent(gap, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode DM lane gap: %w", err)
		}
		raw = append(raw, '\n')
		path := filepath.Join(dir, "dm-gap-"+strings.TrimPrefix(gap.ID, "sha256:")+".json")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			if !os.IsExist(err) {
				return nil, fmt.Errorf("persist DM lane gap %s: %w", gap.ID, err)
			}
			existing, readErr := os.ReadFile(path)
			if readErr != nil || string(existing) != string(raw) {
				return nil, fmt.Errorf("DM lane gap identity collision or unreadable existing record %s", gap.ID)
			}
			paths = append(paths, path)
			continue
		}
		if _, err := f.Write(raw); err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			_ = os.Remove(path)
			return nil, fmt.Errorf("write DM lane gap: %w", err)
		}
		if closeErr != nil {
			_ = os.Remove(path)
			return nil, fmt.Errorf("close DM lane gap: %w", closeErr)
		}
		paths = append(paths, path)
	}
	return paths, nil
}
