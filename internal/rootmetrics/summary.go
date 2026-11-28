package rootmetrics

// This reader is for forensic reporting only. Activation and recovery must not
// consult it: observations are lossy and subject to retention.
import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Summary struct {
	StorePresent          bool           `json:"store_present"`
	Files                 int            `json:"files"`
	Valid                 int            `json:"valid_records"`
	Malformed             int            `json:"malformed_or_unreadable"`
	Unsupported           int            `json:"unsupported_schema"`
	Incomplete            int            `json:"without_terminal"`
	Kinds                 map[string]int `json:"records_by_kind"`
	Terminals             map[string]int `json:"terminal_events_by_kind_and_reason"`
	Stages                map[string]int `json:"events_by_stage_and_reason"`
	Attributions          map[string]int `json:"records_by_attribution"`
	ChangedOwners         map[string]int `json:"records_by_changed_owner"`
	InvalidAttribution    int            `json:"records_with_invalid_attribution"`
	Operations            int            `json:"distinct_observed_operations"`
	OperationOutcomes     map[string]int `json:"distinct_operations_by_observed_outcome"`
	ConflictingOperations int            `json:"operations_with_conflicting_outcomes"`
	First                 *time.Time     `json:"first_event,omitempty"`
	Last                  *time.Time     `json:"last_event,omitempty"`
}

// Summarize reads only bounded regular files with observation filenames. It
// reports counts rather than rates because missing writes cannot be measured.
func Summarize(dir string) (Summary, error) {
	s := Summary{Kinds: map[string]int{}, Terminals: map[string]int{}, Stages: map[string]int{}, Attributions: map[string]int{}, ChangedOwners: map[string]int{}, OperationOutcomes: map[string]int{}}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	s.StorePresent = true
	operations := map[string]map[string]bool{}
	for _, entry := range entries {
		if !attemptFileName.MatchString(entry.Name()) {
			continue
		}
		s.Files++
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
			s.Malformed++
			continue
		}
		f, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			s.Malformed++
			continue
		}
		opened, statErr := f.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			_ = f.Close()
			s.Malformed++
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, 65537))
		_ = f.Close()
		if err != nil || len(data) > 65536 {
			s.Malformed++
			continue
		}
		var r rootAttemptRecord
		if json.Unmarshal(data, &r) != nil {
			s.Malformed++
			continue
		}
		if r.Schema != rootAttemptSchema {
			s.Unsupported++
			continue
		}
		if r.ID+".json" != entry.Name() || (r.Kind != "install" && r.Kind != "rebaseline" && r.Kind != "activation_observation" && r.Kind != "bundle_publication") || len(r.Events) == 0 || len(r.Events) > 64 {
			s.Malformed++
			continue
		}
		valid := true
		for _, e := range r.Events {
			if e.At.IsZero() || !rootAttemptToken(e.Stage) || (e.Reason != "" && !rootAttemptToken(e.Reason)) {
				valid = false
				break
			}
		}
		if !valid {
			s.Malformed++
			continue
		}
		s.Valid++
		s.Kinds[r.Kind]++
		attribution := r.Attribution
		if attribution == "" {
			attribution = "unobserved"
		}
		validAttribution := rootAttemptToken(attribution) && len(r.Owners) <= 256
		seenOwners := map[string]bool{}
		for _, owner := range r.Owners {
			if !ownerName.MatchString(owner) || seenOwners[owner] {
				validAttribution = false
			}
			seenOwners[owner] = true
		}
		if len(r.Owners) > 0 && attribution != "compared" && attribution != "compared_toolchain_changed" {
			validAttribution = false
		}
		if validAttribution {
			s.Attributions[attribution]++
			for owner := range seenOwners {
				s.ChangedOwners[owner]++
			}
		} else {
			s.InvalidAttribution++
		}
		terminal := false
		for _, e := range r.Events {
			if s.First == nil || e.At.Before(*s.First) {
				v := e.At
				s.First = &v
			}
			if s.Last == nil || e.At.After(*s.Last) {
				v := e.At
				s.Last = &v
			}
			key := e.Stage
			if e.Reason != "" {
				key += "/" + e.Reason
			}
			s.Stages[key]++
			if e.Stage == "terminal" {
				terminal = true
				s.Terminals[r.Kind+"/"+e.Reason]++
			}
		}
		if !terminal {
			s.Incomplete++
		}
		if len(r.OperationID) == 32 && rootAttemptHash(r.OperationID+r.OperationID) != "" {
			if operations[r.OperationID] == nil {
				operations[r.OperationID] = map[string]bool{}
			}
			if r.Kind == "activation_observation" {
				for _, e := range r.Events {
					if e.Stage == "terminal" && e.Reason != "" {
						operations[r.OperationID][e.Reason] = true
					}
				}
			}
		}
	}
	s.Operations = len(operations)
	for _, outcomes := range operations {
		if len(outcomes) == 0 {
			s.OperationOutcomes["unobserved"]++
			continue
		}
		if len(outcomes) > 1 {
			s.ConflictingOperations++
			continue
		}
		keys := make([]string, 0, len(outcomes))
		for k := range outcomes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		s.OperationOutcomes[keys[0]]++
	}
	return s, nil
}
