// Package planrepair persists Swarm's narrow authority record for a declared
// plan-repair workflow. Agent Manager owns execution; this package records only
// the immutable authorization frontier and exactly-once application state.
package planrepair

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ApplyState string

const (
	ApplyPending  ApplyState = "pending"
	ApplyClaimed  ApplyState = "claimed"
	ApplyComplete ApplyState = "complete"
)

type Record struct {
	ID                string     `json:"id"`
	EntityKind        string     `json:"entityKind"`
	EntityName        string     `json:"entityName"`
	EntityVersion     string     `json:"entityVersion"`
	PlanReference     string     `json:"planReference"`
	FrontierDigest    string     `json:"frontierDigest"`
	WorkflowExecution string     `json:"workflowExecution"`
	WorkflowDigest    string     `json:"workflowDigest"`
	ApplyState        ApplyState `json:"applyState"`
	AppliedPlanID     string     `json:"appliedPlanId,omitempty"`
}

func (r Record) Validate() error {
	for name, value := range map[string]string{"id": r.ID, "entityKind": r.EntityKind, "entityName": r.EntityName, "entityVersion": r.EntityVersion, "planReference": r.PlanReference, "frontierDigest": r.FrontierDigest, "workflowExecution": r.WorkflowExecution, "workflowDigest": r.WorkflowDigest} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if r.ApplyState != ApplyPending && r.ApplyState != ApplyClaimed && r.ApplyState != ApplyComplete {
		return fmt.Errorf("invalid applyState %q", r.ApplyState)
	}
	if r.ApplyState == ApplyComplete && strings.TrimSpace(r.AppliedPlanID) == "" {
		return fmt.Errorf("appliedPlanId is required after apply")
	}
	return nil
}

type Store struct{ path string }

func NewStore(path string) *Store { return &Store{path: path} }

func (s *Store) Load(id string) (Record, error) {
	records, err := s.loadAll()
	if err != nil {
		return Record{}, err
	}
	r, ok := records[strings.TrimSpace(id)]
	if !ok {
		return Record{}, os.ErrNotExist
	}
	return r, nil
}

func (s *Store) Save(record Record) error {
	if err := record.Validate(); err != nil {
		return err
	}
	records, err := s.loadAll()
	if err != nil {
		return err
	}
	records[record.ID] = record
	return s.saveAll(records)
}

func (s *Store) loadAll() (map[string]Record, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return map[string]Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode plan repair ledger: %w", err)
	}
	out := make(map[string]Record, len(records))
	for _, r := range records {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		if _, exists := out[r.ID]; exists {
			return nil, fmt.Errorf("duplicate repair record %q", r.ID)
		}
		out[r.ID] = r
	}
	return out, nil
}

func (s *Store) saveAll(records map[string]Record) error {
	values := make([]Record, 0, len(records))
	for _, r := range records {
		values = append(values, r)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}
