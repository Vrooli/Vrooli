// Package tasks provides the kernel-private checkpoint bridge. Domain effects
// still use the ordinary binding bridge and the original session's grants.
package tasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
	"program-runtime/internal/tasks"
)

type Bridge struct {
	Store     *tasks.Store
	Contracts *contracts.Index
	Library   *library.Repository
	Sessions  *sessions.Manager
	Programs  *programs.Service
}
type request struct {
	ResumeToken  string         `json:"resume_token"`
	SessionID    string         `json:"session_id"`
	ProgramID    string         `json:"program_id"`
	Operation    string         `json:"operation"`
	Digest       string         `json:"digest"`
	TaskID       string         `json:"task_id"`
	AttemptID    string         `json:"attempt_id"`
	Inputs       map[string]any `json:"inputs"`
	Prepare      map[string]any `json:"prepare"`
	Result       map[string]any `json:"result"`
	FinishInputs map[string]any `json:"finish_inputs"`
	FinishDigest string         `json:"finish_digest"`
	Outcome      string         `json:"outcome"`
}

var validReceipt = regexp.MustCompile(`^prt_resume_v1_[A-Za-z0-9_-]{43}$`)

func (b *Bridge) resolve(ctx context.Context, name, digest string) (contracts.Contract, error) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return contracts.Contract{}, errors.New("operation must be scenario.program")
	}
	c, ok := b.Contracts.Get(parts[0], parts[1])
	if !ok || (digest != "" && c.Digest != digest) {
		if digest == "" {
			return c, errors.New("operation is not registered")
		}
		var err error
		c, err = b.Library.GetDeclaredArtifact(ctx, name, digest)
		if err != nil {
			return c, err
		}
	}
	if c.ValidationError != "" {
		return c, errors.New(c.ValidationError)
	}
	return c, b.Library.RetainDeclared(ctx, c)
}

func (b *Bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req request
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := b.Sessions.Get(r.Context(), req.SessionID); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	p, err := b.Programs.Get(r.Context(), req.ProgramID)
	if err != nil || p.GetSessionId() != req.SessionID {
		writeError(w, http.StatusForbidden, errors.New("program does not belong to session"))
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/internal/program-runtime/tasks/")
	if action == "resolve" {
		c, err := b.resolve(r.Context(), req.Operation, req.Digest)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		writeJSON(w, Spec(c))
		return
	}
	if action == "begin" {
		if !validReceipt.MatchString(req.ResumeToken) {
			writeError(w, 400, errors.New("bounded resume receipt required"))
			return
		}
		for _, id := range []string{req.TaskID, req.AttemptID} {
			if len(id) == 0 || len(id) > 128 {
				writeError(w, 400, errors.New("bounded task_id and attempt_id required"))
				return
			}
		}
		c, err := b.resolve(r.Context(), req.Operation, req.Digest)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		if c.LearningTask == nil {
			writeError(w, 409, errors.New("operation has no learning_task registration"))
			return
		}
		inputs, err := c.ResolveInputs(req.Inputs)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		contextValues := map[string]any{}
		for _, key := range c.LearningTask.ContextFields {
			value, ok := inputs[key]
			if !ok {
				writeError(w, 400, fmt.Errorf("learning context input %q missing", key))
				return
			}
			contextValues[key] = value
		}
		encoded, _ := json.Marshal(contextValues)
		hash := sha256.Sum256(encoded)
		encodedInputs, _ := json.Marshal(inputs)
		inputHash := sha256.Sum256(encodedInputs)
		finish, err := b.resolve(r.Context(), "vrooli-memory.finish-attempt", req.FinishDigest)
		if err != nil {
			writeError(w, 409, fmt.Errorf("pin finish program: %w", err))
			return
		}
		rec, created, err := b.Store.Begin(r.Context(), tasks.Record{InputHash: hex.EncodeToString(inputHash[:]), ResumeHash: receiptHash(req.ResumeToken), TaskID: req.TaskID, AttemptID: req.AttemptID, SessionID: req.SessionID, ProgramID: req.ProgramID, Operation: c.ID, Digest: c.Digest, Scope: c.LearningTask.Scope, ContextKey: "ctx-v1:" + hex.EncodeToString(hash[:]), Provenance: strings.ToLower(strings.TrimPrefix(p.GetProvenance().String(), "PROVENANCE_")), StartedAt: time.Now().UTC().Format(time.RFC3339Nano), FinishDigest: finish.Digest})
		if err != nil {
			writeError(w, 409, err)
			return
		}
		writeJSON(w, map[string]any{"record": publicRecord(rec), "created": created})
		return
	}
	rec, err := b.Store.Get(r.Context(), req.AttemptID)
	if err != nil {
		writeError(w, 404, err)
		return
	}
	authorized := rec.ProgramID == req.ProgramID || (len(req.ResumeToken) >= 32 && rec.ResumeHash == receiptHash(req.ResumeToken) && rec.Provenance == strings.ToLower(strings.TrimPrefix(p.GetProvenance().String(), "PROVENANCE_")))
	if !authorized {
		writeError(w, 403, errors.New("valid resume receipt and unchanged provenance required"))
		return
	}
	if action == "get" {
		writeJSON(w, publicRecord(rec))
		return
	}
	if action == "resume" {
		rec, err = b.Store.Update(r.Context(), req.AttemptID, func(current *tasks.Record) error {
			if current.State != "completed" || current.FinishInputs == nil {
				return errors.New("no completed domain checkpoint; inspect uncertain effects")
			}
			if current.Provenance != "operator" && current.Provenance != "test" {
				return errors.New("memory provenance unsupported")
			}
			if current.Delivery == "delivered" || current.Delivery == "pending" {
				return nil
			}
			if current.LastError == "advice_decision_required" {
				return errors.New("finish requires explicit advice decisions")
			}
			current.Delivery = "pending"
			current.DeliveryAttempts = 0
			current.NextAttemptAt = time.Now().UTC().Format(time.RFC3339Nano)
			return nil
		})
		if err != nil {
			writeError(w, 409, err)
			return
		}
		writeJSON(w, publicRecord(rec))
		return
	}
	// Reads have the same visibility as persisted programs. Completing a running
	// attempt requires its admitting program. An explicit advice-only finalization
	// may use a fresh session after the original session has been reclaimed.
	adviceOnly := action == "complete" && rec.State == "completed" && rec.Delivery == "blocked" && rec.LastError == "advice_decision_required"
	if rec.ProgramID != req.ProgramID && !adviceOnly {
		writeError(w, 403, errors.New("attempt checkpoints belong to their admitting program"))
		return
	}
	if adviceOnly && (rec.ResumeHash != receiptHash(req.ResumeToken) || rec.Provenance != strings.ToLower(strings.TrimPrefix(p.GetProvenance().String(), "PROVENANCE_"))) {
		writeError(w, 403, errors.New("valid resume receipt and unchanged provenance required"))
		return
	}
	rec, err = b.Store.Update(r.Context(), req.AttemptID, func(current *tasks.Record) error {
		switch action {
		case "start":
			if current.State != "prepared" {
				return errors.New("domain dispatch already admitted; inspect attempt")
			}
			current.Prepare = req.Prepare
			current.State = "running"
		case "complete":
			if current.State != "running" && !adviceOnly {
				return errors.New("attempt is not running")
			}
			if req.Result == nil || req.FinishInputs == nil {
				return errors.New("completed domain result and exact finish inputs required")
			}
			if adviceOnly && !reflect.DeepEqual(current.Result, req.Result) {
				return errors.New("completed domain result is immutable")
			}
			c, err := b.resolve(r.Context(), current.Operation, current.Digest)
			if err != nil {
				return err
			}
			if err = validateCompletion(current, c, req); err != nil {
				return err
			}
			current.Result = req.Result
			current.FinishInputs = req.FinishInputs
			current.Outcome = req.Outcome
			current.State = "completed"
			current.Delivery = "pending"
			current.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			current.LastError = ""
			attempt, _ := req.FinishInputs["attempt"].(map[string]any)
			if attempt["recall_status"] == nil {
				current.Delivery = "blocked"
				current.LastError = "advice_decision_required"
			}
			if current.Provenance != "operator" && current.Provenance != "test" {
				current.Delivery = "blocked"
				current.LastError = "memory_provenance_unsupported"
			}
			current.NextAttemptAt = current.FinishedAt
		default:
			return errors.New("unknown task checkpoint")
		}
		return nil
	})
	if err != nil {
		writeError(w, 409, err)
		return
	}
	writeJSON(w, publicRecord(rec))
}
func Spec(c contracts.Contract) map[string]any {
	return map[string]any{"name": c.Name, "scenario": c.Scenario, "contract": true, "current": true, "declaration": json.RawMessage(c.Declaration), "digest": c.Digest, "source": c.Source}
}
func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
func writeJSON(w http.ResponseWriter, value any) { _ = json.NewEncoder(w).Encode(value) }
func receiptHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func publicRecord(r *tasks.Record) map[string]any {
	raw, _ := json.Marshal(r)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	delete(out, "resume_hash")
	return out
}

func valueAt(v any, path string) any {
	for _, part := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[part]
	}
	return v
}

func validateCompletion(record *tasks.Record, c contracts.Contract, req request) error {
	if c.LearningTask == nil {
		return errors.New("missing registered learning boundary")
	}
	status, _ := valueAt(req.Result, c.LearningTask.Outcome.StatusPath).(string)
	expected := c.LearningTask.Outcome.Mapping[status]
	if expected == "" {
		expected = "unknown"
	}
	var refs []any
	seen := map[string]bool{}
	for _, path := range c.LearningTask.Outcome.EvidencePaths {
		values, _ := valueAt(req.Result, path).([]any)
		for _, raw := range values {
			v, ok := raw.(string)
			if ok && strings.TrimSpace(v) != "" && len(v) <= 512 && !seen[v] && len(refs) < 20 {
				seen[v] = true
				refs = append(refs, v)
			}
		}
	}
	if expected == "verified_success" && len(refs) == 0 {
		expected = "unknown"
	}
	if req.Outcome != expected {
		return errors.New("outcome must follow the registered operation's explicit signal and evidence")
	}
	attempt, ok := req.FinishInputs["attempt"].(map[string]any)
	if !ok || req.FinishInputs["scope"] != record.Scope {
		return errors.New("capture scope or attempt missing")
	}
	for field, expected := range map[string]string{"attempt_id": record.AttemptID, "task_id": record.TaskID, "context_key": record.ContextKey, "provenance": record.Provenance, "operation": c.LearningTask.Operation, "started_at": record.StartedAt, "task_started_at": record.TaskStartedAt, "outcome": req.Outcome} {
		if attempt[field] != expected {
			return fmt.Errorf("capture %s cannot replace runtime identity or outcome", field)
		}
	}
	if attempt["attempt_number"] != float64(record.AttemptNumber) {
		return errors.New("capture attempt ordinal cannot change")
	}
	evidence, _ := attempt["evidence_refs"].([]any)
	if len(evidence) != len(refs) || (len(refs) > 0 && !reflect.DeepEqual(evidence, refs)) {
		return errors.New("capture evidence must come from declared result paths")
	}
	if record.State == "completed" {
		old, _ := record.FinishInputs["attempt"].(map[string]any)
		for k, v := range old {
			if k != "advice" && k != "recall_status" && !reflect.DeepEqual(v, attempt[k]) {
				return errors.New("capture finalization may only add advice decisions")
			}
		}
	}
	return nil
}
