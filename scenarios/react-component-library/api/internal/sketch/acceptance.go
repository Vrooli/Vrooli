package sketch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/platform-go"
)

const AcceptancePolicy = "design-acceptance/1"

var ErrAcceptanceConflict = errors.New("acceptance idempotency key already names different inputs")

// AcceptanceIntent records caller-declared identity, not authenticated human
// approval. No caller-supplied verdict or proof boolean crosses this boundary.
type AcceptanceIntent struct {
	DesignID           string   `json:"designId"`
	CandidateHash      string   `json:"candidateHash"`
	ExpectedRenderHash string   `json:"expectedRenderHash"`
	Actor              string   `json:"actor"`
	Kit                string   `json:"kit"`
	Theme              string   `json:"theme"`
	Direction          string   `json:"direction"`
	PreviewState       string   `json:"previewState"`
	MissingLabel       string   `json:"missingLabel"`
	FailedLabel        string   `json:"failedLabel"`
	CritiqueIDs        []string `json:"critiqueIds"`
}
type AcceptanceRequirement struct {
	Code        string   `json:"code"`
	Status      string   `json:"status"`
	Detail      string   `json:"detail"`
	EvidenceIDs []string `json:"evidenceIds,omitempty"`
}
type AcceptanceFacts struct {
	PolicyVersion string                  `json:"policyVersion"`
	CandidateHash string                  `json:"candidateHash"`
	RenderHash    string                  `json:"renderHash"`
	InputHashes   map[string]string       `json:"inputHashes"`
	Requirements  []AcceptanceRequirement `json:"requirements"`
}
type AcceptanceDecision struct {
	SchemaVersion int              `json:"schemaVersion"`
	ID            string           `json:"id"`
	Hash          string           `json:"hash"`
	State         string           `json:"state"` // prepared, accepted, needs_evidence, failed
	Intent        AcceptanceIntent `json:"intent"`
	BaseHash      string           `json:"baseHash"`
	Facts         *AcceptanceFacts `json:"facts,omitempty"`
	Detail        string           `json:"detail,omitempty"`
	CreatedAt     string           `json:"createdAt"`
	CompletedAt   string           `json:"completedAt,omitempty"`
}

func validateAcceptanceIntent(i AcceptanceIntent) error {
	if !safeSegment.MatchString(i.DesignID) || !validHash.MatchString(i.CandidateHash) || !validHash.MatchString(i.ExpectedRenderHash) || strings.TrimSpace(i.Actor) == "" || len(i.Actor) > 200 || len(i.CritiqueIDs) > 16 {
		return errors.New("exact candidate/render, design identity and bounded declared actor are required")
	}
	for _, s := range []string{i.Kit, i.Theme, i.Direction, i.PreviewState, i.MissingLabel, i.FailedLabel} {
		if len(s) > 1000 {
			return errors.New("acceptance render input exceeds limits")
		}
	}
	seen := map[string]bool{}
	for _, id := range i.CritiqueIDs {
		if strings.TrimSpace(id) == "" || len(id) > 200 || seen[id] {
			return errors.New("critique IDs must be bounded and unique")
		}
		seen[id] = true
	}
	return nil
}
func acceptancePath(designID, id string) (string, error) {
	if !safeSegment.MatchString(designID) || !strings.HasPrefix(id, "acceptance_") || !validHash.MatchString(strings.TrimPrefix(id, "acceptance_")) {
		return "", errors.New("invalid acceptance identity")
	}
	return filepath.Join("designs", designID, "acceptance", id+".json"), nil
}
func acceptanceID(key string) (string, error) {
	if strings.TrimSpace(key) == "" || len(key) > 200 {
		return "", errors.New("bounded acceptance idempotency key is required")
	}
	hash := sha256.Sum256([]byte(key))
	return "acceptance_" + hex.EncodeToString(hash[:]), nil
}
func decisionHash(d AcceptanceDecision) (string, error) {
	d.Hash = ""
	raw, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func writeAcceptance(root *os.Root, d AcceptanceDecision) (AcceptanceDecision, error) {
	var err error
	d.Hash, err = decisionHash(d)
	if err != nil {
		return d, err
	}
	path, err := acceptancePath(d.Intent.DesignID, d.ID)
	if err != nil {
		return d, err
	}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return d, err
	}
	return d, storage.WriteFileAtomicInRoot(root, path, raw, 0600)
}
func readAcceptance(root *os.Root, designID, id string) (AcceptanceDecision, error) {
	var d AcceptanceDecision
	path, err := acceptancePath(designID, id)
	if err != nil {
		return d, err
	}
	raw, err := root.ReadFile(path)
	if err != nil {
		return d, err
	}
	if len(raw) > 1024*1024 {
		return d, errors.New("acceptance record exceeds size limit")
	}
	if err = rejectDuplicateKeys(raw); err != nil {
		return d, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&d); err != nil {
		return d, err
	}
	if err = validateAcceptanceIntent(d.Intent); err != nil {
		return d, err
	}
	hash, err := decisionHash(d)
	if err != nil || d.SchemaVersion != 1 || d.Hash != hash || d.ID != id || d.Intent.DesignID != designID || !validHash.MatchString(d.BaseHash) {
		return d, errors.New("acceptance record identity mismatch")
	}
	switch d.State {
	case "prepared", "accepted", "needs_evidence", "failed":
	default:
		return d, errors.New("invalid acceptance decision state")
	}
	if d.State == "accepted" && (d.Facts == nil || !acceptanceReady(d.Intent, *d.Facts)) {
		return d, errors.New("accepted record does not satisfy its policy")
	}
	return d, nil
}
func (s *Store) ReadAcceptance(scenario, designID, id string) (AcceptanceDecision, error) {
	root, err := s.openExperience(scenario)
	if err != nil {
		return AcceptanceDecision{}, err
	}
	defer root.Close()
	return readAcceptance(root, designID, id)
}

// acceptanceReady is a conjunction, not an average. Unknown, omitted or
// contradictory requirements cannot accidentally become permission to apply.
func acceptanceReady(intent AcceptanceIntent, f AcceptanceFacts) bool {
	if f.PolicyVersion != AcceptancePolicy || f.CandidateHash != intent.CandidateHash || f.RenderHash != intent.ExpectedRenderHash || len(f.InputHashes) == 0 {
		return false
	}
	for _, key := range []string{"appearance", "bindings", "composition", "dependencies", "fixtures", "harness", "bundle"} {
		if !validHash.MatchString(f.InputHashes[key]) {
			return false
		}
	}
	for _, hash := range f.InputHashes {
		if !validHash.MatchString(hash) {
			return false
		}
	}
	required := map[string]bool{"page_freshness": false, "render_freshness": false, "render_completeness": false, "visual_review": false, "behavior_evidence": false, "rubric_calibration": false, "independent_review": false}
	for _, r := range f.Requirements {
		if r.Status != "passed" || len(r.EvidenceIDs) == 0 {
			return false
		}
		for _, id := range r.EvidenceIDs {
			if strings.TrimSpace(id) == "" {
				return false
			}
		}
		if _, ok := required[r.Code]; ok {
			required[r.Code] = true
		}
	}
	for _, passed := range required {
		if !passed {
			return false
		}
	}
	return true
}

// RecordAcceptance persists prepared intent before consulting proof producers.
// The per-key lock serializes retries; after a crash the same prepared intent
// can resume. Terminal records are immutable and replay without producer calls.
func (s *Store) RecordAcceptance(ctx context.Context, scenario, key string, intent AcceptanceIntent, evaluate func(context.Context) (AcceptanceFacts, error)) (AcceptanceDecision, error) {
	if err := validateAcceptanceIntent(intent); err != nil {
		return AcceptanceDecision{}, err
	}
	id, err := acceptanceID(key)
	if err != nil {
		return AcceptanceDecision{}, err
	}
	if evaluate == nil {
		return AcceptanceDecision{}, errors.New("acceptance evidence evaluator required")
	}
	root, err := s.openExperience(scenario)
	if err != nil {
		return AcceptanceDecision{}, err
	}
	defer root.Close()
	lock, err := root.OpenFile(filepath.Join("pages", "."+id+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return AcceptanceDecision{}, err
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, false)
	if err != nil {
		return AcceptanceDecision{}, err
	}
	defer release()
	d, err := readAcceptance(root, intent.DesignID, id)
	if err == nil {
		if !reflect.DeepEqual(d.Intent, intent) {
			return d, ErrAcceptanceConflict
		}
		if d.State != "prepared" {
			return d, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return d, err
	} else {
		c, e := readCandidate(root, intent.DesignID, intent.CandidateHash)
		if e != nil {
			return d, e
		}
		d = AcceptanceDecision{SchemaVersion: 1, ID: id, State: "prepared", Intent: intent, BaseHash: c.BaseHash, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		d, err = writeAcceptance(root, d)
		if err != nil {
			return d, err
		}
	}
	if err = ctx.Err(); err != nil {
		return d, err
	}
	facts, e := evaluate(ctx)
	if e != nil {
		d.State = "failed"
		d.Detail = e.Error()
	} else {
		d.Facts = &facts
		d.State = "needs_evidence"
		if acceptanceReady(intent, facts) {
			// Protect the final base-page check and decision publication from authored
			// page writes. Immutable asset/render inputs remain fixed in Facts.
			c, e := readCandidate(root, intent.DesignID, intent.CandidateHash)
			if e != nil {
				return d, e
			}
			pageLock, e := root.OpenFile(filepath.Join("pages", "."+c.Page+".lock"), os.O_CREATE|os.O_RDWR, 0600)
			if e != nil {
				return d, e
			}
			defer pageLock.Close()
			unlock, e := platform.LockFile(pageLock, false)
			if e != nil {
				return d, e
			}
			defer unlock()
			current, e := s.Read(scenario, c.Page)
			if e != nil {
				return d, e
			}
			if current.ContentHash == d.BaseHash {
				d.State = "accepted"
			} else {
				d.Detail = "Authored page changed during evidence evaluation."
				d.Facts.Requirements = append(d.Facts.Requirements, AcceptanceRequirement{Code: "page_freshness", Status: "blocked", Detail: d.Detail})
			}
		}
	}
	d.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if len(d.Detail) > 4000 {
		d.Detail = d.Detail[:4000]
	}
	if d.Facts != nil && len(d.Facts.Requirements) > 256 {
		return d, fmt.Errorf("acceptance facts exceed requirement limit")
	}
	return writeAcceptance(root, d)
}
