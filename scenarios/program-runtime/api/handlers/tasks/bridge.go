// Package tasks provides the kernel-private checkpoint bridge. Domain effects
// still use the ordinary binding bridge and the original session's grants.
package tasks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	scenariomanifest "github.com/vrooli/vrooli/packages/scenario-manifest"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
	"program-runtime/internal/tasks"
)

type Bridge struct {
	publicationMu sync.Mutex
	Store         *tasks.Store
	Contracts     *contracts.Index
	Library       *library.Repository
	Sessions      *sessions.Manager
	Programs      *programs.Service
	RepoRoot      string
}
type request struct {
	ClaimRef            string         `json:"claim_ref"`
	FeedbackRef         string         `json:"feedback_ref"`
	StepAttemptID       string         `json:"step_attempt_id"`
	Verb                string         `json:"verb"`
	Name                string         `json:"name"`
	Artifact            map[string]any `json:"artifact"`
	FragmentHash        string         `json:"fragment_hash"`
	CompatibilityDigest string         `json:"compatibility_digest"`
	ObservationID       string         `json:"observation_id"`
	Disposition         string         `json:"disposition"`
	Dimension           string         `json:"dimension"`
	Correction          string         `json:"correction"`
	Owner               string         `json:"owner"`
	FindingID           string         `json:"finding_id"`
	ExpectedState       string         `json:"expected_state"`
	NextState           string         `json:"next_state"`
	MinContexts         int            `json:"min_contexts"`
	MaxAgeDays          float64        `json:"max_age_days"`
	CacheHit            bool           `json:"cache_hit"`
	Baseline            map[string]any `json:"baseline"`
	ContextKey          string         `json:"context_key"`
	InputDigest         string         `json:"input_digest"`
	Compatibility       map[string]any `json:"compatibility"`
	Evidence            []string       `json:"evidence"`
	ResumeToken         string         `json:"resume_token"`
	SessionID           string         `json:"session_id"`
	ProgramID           string         `json:"program_id"`
	Operation           string         `json:"operation"`
	Scope               string         `json:"scope"`
	Digest              string         `json:"digest"`
	TaskID              string         `json:"task_id"`
	AttemptID           string         `json:"attempt_id"`
	Inputs              map[string]any `json:"inputs"`
	Prepare             map[string]any `json:"prepare"`
	Result              map[string]any `json:"result"`
	FinishInputs        map[string]any `json:"finish_inputs"`
	FinishDigest        string         `json:"finish_digest"`
	Outcome             string         `json:"outcome"`
	StepKey             string         `json:"step_key"`
	Fragment            string         `json:"fragment"`
	VerifiedDelta       int            `json:"verified_delta"`
	ContradictedDelta   int            `json:"contradicted_delta"`
	Source              string         `json:"source"`
	SourceProgramID     string         `json:"source_program_id"`
	StepName            string         `json:"step_name"`
	TraceInputs         map[string]any `json:"trace_inputs"`
	TraceOutput         map[string]any `json:"trace_output"`
	MinVerified         int            `json:"min_verified"`
	PromoteName         string         `json:"promote_name"`
}

func taskScope(c contracts.Contract, requested string) (string, error) {
	defaultScope := c.Scenario + "-usage"
	if c.LearningTask != nil && c.LearningTask.Scope != "" {
		defaultScope = c.LearningTask.Scope
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return defaultScope, nil
	}
	requestedScenario := strings.TrimSuffix(requested, "-usage")
	canonicalScenario := map[string]string{"bas": "browser-automation-studio"}[requestedScenario]
	if canonicalScenario == "" {
		canonicalScenario = requestedScenario
	}
	if canonicalScenario == c.Scenario {
		return requested, nil
	}
	for _, binding := range c.Bindings {
		if binding.ID == requestedScenario+"/learning/record" && binding.Effect == "write" {
			return requested, nil
		}
	}
	return "", fmt.Errorf("foreign learning scope %q requires a declared write binding", requested)
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

// markPublishedBaselines flags each fragment its owning contract already declares as a reviewed
// learning.baselines entry, so a promotion-candidate count excludes work that is already done.
// Publication copies the verified fragment text verbatim, so an exact (step name, fragment) match
// still identifies it after the program source is edited and fragment.Source no longer agrees.
func markPublishedBaselines(declared []contracts.Contract, fragments []tasks.Fragment) {
	published := map[[2]string]bool{}
	for _, contract := range declared {
		var declaration struct {
			Learning struct {
				Baselines map[string]struct {
					Fragment string `json:"fragment"`
				} `json:"baselines"`
			} `json:"learning"`
		}
		if len(contract.Declaration) == 0 || json.Unmarshal(contract.Declaration, &declaration) != nil {
			continue
		}
		for step, baseline := range declaration.Learning.Baselines {
			if text := strings.TrimSpace(baseline.Fragment); text != "" {
				published[[2]string{step, text}] = true
			}
		}
	}
	for i := range fragments {
		fragments[i].Published = published[[2]string{fragments[i].StepName, strings.TrimSpace(fragments[i].Fragment)}]
	}
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

	provenance := strings.ToLower(strings.TrimPrefix(p.GetProvenance().String(), "PROVENANCE_"))
	if action == "learning_result_put" {
		rec, err := b.Store.Get(r.Context(), req.AttemptID)
		if err != nil || (rec.ProgramID != req.ProgramID && rec.ResumeHash != receiptHash(req.ResumeToken)) || rec.TaskID != req.TaskID || !sameLearningCohort(rec.Provenance, provenance) || rec.Scope != req.Scope {
			writeError(w, 403, errors.New("learning result must belong to admitting task"))
			return
		}
		if req.StepKey != "" {
			if err := b.Store.ClaimFragmentCohort(r.Context(), req.StepKey, provenance); err != nil {
				writeError(w, 403, err)
				return
			}
		}
		owner := strings.TrimSuffix(req.Scope, "-usage")
		if value, ok := req.Artifact["owner"].(string); ok && value != "" {
			owner = value
		}
		if owner == "bas" {
			owner = "browser-automation-studio"
		}
		if owner == "" || strings.ContainsAny(owner, "/\\.") {
			owner = "program-runtime"
		}
		if b.RepoRoot != "" {
			if info, err := os.Stat(filepath.Join(b.RepoRoot, "scenarios", owner)); err != nil || !info.IsDir() {
				owner = "program-runtime"
			}
		}
		err = b.Store.PutLearningResult(r.Context(), tasks.LearningResult{Owner: owner, FeedbackRef: req.FeedbackRef, TaskID: req.TaskID, AttemptID: req.AttemptID, StepAttemptID: req.StepAttemptID, Scope: req.Scope, Verb: req.Verb, Name: req.Name, Artifact: req.Artifact, StepKey: req.StepKey, FragmentHash: req.FragmentHash, StepName: req.StepName, CompatibilityDigest: req.CompatibilityDigest, Provenance: rec.Provenance})
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, map[string]any{"stored": true, "feedback_ref": req.FeedbackRef})
		return
	}
	if action == "learning_result_status" {
		result, err := b.Store.LearningResultStatus(r.Context(), req.FeedbackRef, req.Artifact, provenance)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, result)
		return
	}
	if action == "learning_feedback" {
		finding, err := b.Store.ApplyFeedback(r.Context(), tasks.Feedback{ObservationID: req.ObservationID, FeedbackRef: req.FeedbackRef, Disposition: req.Disposition, Dimension: req.Dimension, Evidence: req.Evidence, Correction: req.Correction}, provenance)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		target, _ := b.Store.LearningResultStatus(r.Context(), req.FeedbackRef, nil, provenance)
		writeJSON(w, map[string]any{"stored": true, "finding": finding, "target": target})
		return
	}
	if action == "learning_findings" {
		findings, err := b.Store.ListFindings(r.Context(), req.Owner)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, map[string]any{"findings": findings})
		return
	}
	if action == "learning_finding_transition" {
		finding, err := b.Store.TransitionFinding(r.Context(), req.FindingID, req.ExpectedState, req.NextState, req.Owner, req.ClaimRef, req.Evidence)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		writeJSON(w, map[string]any{"finding": finding})
		return
	}
	if action == "fragment_rejection" {
		if err := b.Store.ClaimFragmentCohort(r.Context(), req.StepKey, provenance); err != nil {
			writeError(w, 403, err)
			return
		}
		rejected, err := b.Store.FragmentRejected(r.Context(), req.StepKey, req.FragmentHash)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, map[string]any{"rejected": rejected})
		return
	}
	if action == "fragment_get" {
		if err := b.Store.ClaimFragmentCohort(r.Context(), req.StepKey, provenance); err != nil {
			writeError(w, 403, err)
			return
		}
		rejected, rejectErr := b.Store.RejectedFragmentHashes(r.Context(), req.StepKey)
		if rejectErr != nil {
			writeError(w, 500, rejectErr)
			return
		}
		fragment, err := b.Store.EligibleFragment(r.Context(), req.StepKey, req.MinVerified, req.MinContexts, req.MaxAgeDays)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, map[string]any{"found": false, "rejected_hashes": rejected, "rejected_truncated": len(rejected) > 100})
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, map[string]any{"found": true, "fragment": fragment, "rejected_hashes": rejected, "rejected_truncated": len(rejected) > 100})
		return
	}
	if action == "fragment_list" {
		fragments, err := b.Store.ListFragments(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if b.Contracts != nil {
			markPublishedBaselines(b.Contracts.List(), fragments)
		}
		writeJSON(w, map[string]any{"fragments": fragments})
		return
	}
	if action == "delivery_metrics" {
		count, err := b.Store.AgedBlocked(r.Context(), time.Now().UTC().Add(-24*time.Hour))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		metrics, err := b.Store.FeedbackDeliveryMetrics(r.Context())
		if err != nil {
			writeError(w, 500, err)
			return
		}
		metrics["blocked_deliveries_aged"] = count
		writeJSON(w, metrics)
		return
	}
	if action == "fragment_put" {
		rec, err := b.Store.Get(r.Context(), req.AttemptID)
		if err != nil || !sameLearningCohort(rec.Provenance, provenance) || (rec.ProgramID != req.ProgramID && (!validReceipt.MatchString(req.ResumeToken) || rec.ResumeHash != receiptHash(req.ResumeToken))) {
			writeError(w, 403, errors.New("fragment evidence belongs to its admitting task"))
			return
		}
		if err := b.Store.ClaimFragmentCohort(r.Context(), req.StepKey, provenance); err != nil {
			writeError(w, 403, err)
			return
		}

		fragment, err := b.Store.PutFragment(r.Context(), tasks.Fragment{StepKey: req.StepKey, Fragment: req.Fragment, Source: req.Source, SourceProgramID: req.SourceProgramID, StepName: req.StepName, TraceInputs: req.TraceInputs, TraceOutput: req.TraceOutput, AttemptID: req.AttemptID, InputDigest: req.InputDigest, Compatibility: req.Compatibility, Evidence: req.Evidence, CacheHit: req.CacheHit}, req.VerifiedDelta, req.ContradictedDelta)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, map[string]any{"found": true, "fragment": fragment})
		return
	}
	if action == "fragment_promote" {
		writeError(w, http.StatusConflict, errors.New("source crystallization retired; publish a reviewed baseline with tasks.fragment_promote"))
		return
	}
	if action == "fragment_publish" {
		if p.GetProvenance().String() != "PROVENANCE_OPERATOR" {
			writeError(w, http.StatusForbidden, errors.New("baseline publication requires operator review"))
			return
		}
		b.publicationMu.Lock()
		defer b.publicationMu.Unlock()
		c, err := b.resolve(r.Context(), req.Operation, req.Digest)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		fragment, err := b.Store.BestFragment(r.Context(), req.StepKey)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		if req.MinVerified < 1 || req.MinVerified > 100 || fragment.Verified < req.MinVerified || fragment.Contexts < 2 || fragment.StepName != req.StepName || fragment.Source != c.Source || fragment.ContradictedSinceEdit != 0 {
			writeError(w, 409, errors.New("fragment is not qualified for publication"))
			return
		}
		if req.Baseline["fragment"] != fragment.Fragment || !reflect.DeepEqual(req.Baseline["compatibility"], fragment.Compatibility) {
			writeError(w, 409, errors.New("baseline does not match verified candidate"))
			return
		}
		if reviewer, ok := req.Baseline["reviewed_by"].(string); !ok || strings.TrimSpace(reviewer) == "" {
			writeError(w, 400, errors.New("reviewer required"))
			return
		}
		if evidence, ok := req.Baseline["evidence"].([]any); !ok || len(evidence) == 0 || len(evidence) > 16 {
			writeError(w, 400, errors.New("review evidence required"))
			return
		}
		if fixtures, ok := req.Baseline["fixtures"].([]any); !ok || len(fixtures) == 0 || len(fixtures) > 16 {
			writeError(w, 400, errors.New("curated fixtures required"))
			return
		}
		asset, assetErr := json.Marshal(req.Baseline)
		if assetErr != nil || len(asset) > 32*1024 {
			writeError(w, 400, errors.New("bounded baseline required"))
			return
		}
		// Publication is a compare-and-swap of one existing declared asset, never arbitrary source.
		path := c.SourcePath
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			writeError(w, 409, errors.New("regular declaration required"))
			return
		}
		current, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(current, c.Declaration) {
			writeError(w, 409, errors.New("declaration changed; prepare again"))
			return
		}
		sourcePath := strings.TrimSuffix(path, filepath.Ext(path)) + ".py"
		sourceInfo, sourceErr := os.Lstat(sourcePath)
		if sourceErr != nil || !sourceInfo.Mode().IsRegular() {
			writeError(w, 409, errors.New("regular program source required"))
			return
		}
		source, sourceErr := os.ReadFile(sourcePath)
		if sourceErr != nil || string(source) != c.Source {
			writeError(w, 409, errors.New("program source changed; prepare again"))
			return
		}
		var declaration map[string]any
		if err = json.Unmarshal(current, &declaration); err != nil {
			writeError(w, 400, err)
			return
		}
		learning, _ := declaration["learning"].(map[string]any)
		if learning == nil {
			learning = map[string]any{}
		}
		baselines, _ := learning["baselines"].(map[string]any)
		if baselines == nil {
			baselines = map[string]any{}
		}
		baselines[req.StepName] = req.Baseline
		learning["baselines"] = baselines
		declaration["learning"] = learning
		encoded, err := json.MarshalIndent(declaration, "", "  ")
		if err != nil || len(encoded) > 1024*1024 {
			writeError(w, 400, errors.New("bounded declaration required"))
			return
		}
		temp, err := os.CreateTemp(filepath.Dir(path), ".baseline-*.tmp")
		if err != nil {
			writeError(w, 500, err)
			return
		}
		defer os.Remove(temp.Name())
		if _, err = temp.Write(append(encoded, '\n')); err == nil {
			err = temp.Sync()
		}
		closeErr := temp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Chmod(temp.Name(), info.Mode().Perm())
		}
		if err == nil {
			err = os.Rename(temp.Name(), path)
		}
		if err != nil {
			writeError(w, 500, err)
			return
		}
		_, err = b.Contracts.Refresh(b.RepoRoot)
		result := map[string]any{"published": true, "operation": req.Operation, "step_name": req.StepName, "digest": req.Baseline["digest"]}
		if err != nil {
			result["refresh_error"] = err.Error()
		}
		writeJSON(w, result)
		return
	}
	if action == "resolve" {
		c, err := b.resolve(r.Context(), req.Operation, req.Digest)
		if err != nil {
			writeError(w, 409, err)
			return
		}
		edge, _ := b.manifestEdge(c.Scenario)
		writeJSON(w, Spec(c, edge))
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
		inputs, err := c.ResolveInputs(req.Inputs)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		contextValues := map[string]any{}
		if c.LearningTask != nil {
			for _, key := range c.LearningTask.ContextFields {
				value, ok := inputs[key]
				if !ok {
					writeError(w, 400, fmt.Errorf("learning context input %q missing", key))
					return
				}
				contextValues[key] = value
			}
		} else {
			contextValues = inputs
		}
		encoded, _ := json.Marshal(contextValues)
		hash := sha256.Sum256(encoded)
		encodedInputs, _ := json.Marshal(inputs)
		inputHash := sha256.Sum256(encodedInputs)
		finish, err := b.resolve(r.Context(), "vrooli-memory.finish-attempt", req.FinishDigest)
		finishDigest := ""
		if err != nil {
			edge, declared := b.manifestEdge(c.Scenario)
			// Memory is optional for contracts without a must_start edge. The
			// checkpoint remains durable and completion records the explicit
			// memory_not_installed delivery state; only an enabled must_start
			// edge makes the missing finish contract an admission failure.
			if declared && edge["startup_policy"] == "must_start" {
				writeError(w, 409, fmt.Errorf("pin finish program: %w", err))
				return
			}
		} else {
			finishDigest = finish.Digest
		}
		scope, err := taskScope(c, req.Scope)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		contextKey := "ctx-v1:" + hex.EncodeToString(hash[:])
		if req.ContextKey != "" {
			if len(req.ContextKey) > 1024 {
				writeError(w, 400, errors.New("bounded context_key required"))
				return
			}
			contextKey = req.ContextKey
		}
		rec, created, err := b.Store.Begin(r.Context(), tasks.Record{InputHash: hex.EncodeToString(inputHash[:]), ResumeHash: receiptHash(req.ResumeToken), TaskID: req.TaskID, AttemptID: req.AttemptID, SessionID: req.SessionID, ProgramID: req.ProgramID, Operation: c.ID, Digest: c.Digest, Scope: scope, ContextKey: contextKey, Provenance: strings.ToLower(strings.TrimPrefix(p.GetProvenance().String(), "PROVENANCE_")), StartedAt: time.Now().UTC().Format(time.RFC3339Nano), FinishDigest: finishDigest})
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
			if current.Provenance != "operator" && current.Provenance != "test" && current.Provenance != "agent" {
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
			if current.Provenance != "operator" && current.Provenance != "test" && current.Provenance != "agent" {
				current.Delivery = "blocked"
				current.LastError = "memory_provenance_unsupported"
			}
			if current.FinishDigest == "" {
				current.Delivery = "blocked"
				current.LastError = "memory_not_installed"
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
	if action == "complete" {
		if raw, ok := req.FinishInputs["attempts"].([]any); ok && len(raw) > 1 {
			nodes := make([]map[string]any, 0, len(raw)-1)
			for _, item := range raw[1:] {
				node, ok := item.(map[string]any)
				if !ok {
					writeError(w, 409, errors.New("attempt tree node must be an object"))
					return
				}
				nodes = append(nodes, node)
			}
			if err := b.Store.InsertChildren(r.Context(), rec, nodes); err != nil {
				writeError(w, 409, err)
				return
			}
		}
	}
	writeJSON(w, publicRecord(rec))
}

func Spec(c contracts.Contract, edge map[string]any) map[string]any {
	result := map[string]any{"name": c.Name, "scenario": c.Scenario, "contract": true, "current": true, "declaration": json.RawMessage(c.Declaration), "digest": c.Digest, "source": c.Source}
	if edge != nil {
		result["manifest_edge"] = edge
	}
	return result
}

func (b *Bridge) manifestEdge(scenario string) (map[string]any, bool) {
	if b.RepoRoot == "" {
		return nil, false
	}
	manifest, err := scenariomanifest.Load(fmt.Sprintf("%s/scenarios/%s/.vrooli/service.json", b.RepoRoot, scenario))
	if err != nil {
		return nil, false
	}
	dependency, ok := manifest.Dependencies.Scenarios["vrooli-memory"]
	if !ok {
		return nil, false
	}
	return map[string]any{"enabled": dependency.Enabled, "required": dependency.Required, "startup_policy": dependency.NormalizedStartupPolicy()}, true
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
		attempt, ok := req.FinishInputs["attempt"].(map[string]any)
		if !ok || req.FinishInputs["scope"] != record.Scope {
			return errors.New("capture scope or attempt missing")
		}
		for field, expected := range map[string]string{"attempt_id": record.AttemptID, "task_id": record.TaskID, "context_key": record.ContextKey, "provenance": record.Provenance, "operation": c.ID, "started_at": record.StartedAt, "task_started_at": record.TaskStartedAt, "outcome": req.Outcome} {
			if attempt[field] != expected {
				return fmt.Errorf("capture %s cannot replace runtime identity or outcome", field)
			}
		}
		if attempt["attempt_number"] != float64(record.AttemptNumber) {
			return errors.New("capture attempt ordinal cannot change")
		}
		if req.Outcome == "verified_success" {
			evidence, _ := attempt["evidence_refs"].([]any)
			if len(evidence) == 0 || len(evidence) > 16 {
				return errors.New("verified_success requires evidence")
			}
		}
		if raw, present := req.FinishInputs["attempts"]; present {
			nodes, ok := raw.([]any)
			if !ok || len(nodes) == 0 || len(nodes) > 32 {
				return errors.New("attempts must contain 1..32 attempt objects")
			}
			known := map[string]bool{}
			for index, rawNode := range nodes {
				node, ok := rawNode.(map[string]any)
				if !ok {
					return errors.New("attempt tree node must be an object")
				}
				id, _ := node["attempt_id"].(string)
				if id == "" || known[id] {
					return errors.New("attempt tree node IDs must be present and unique")
				}
				if index == 0 && id != record.AttemptID {
					return errors.New("attempt tree root must match checkpoint")
				}
				if index > 0 {
					parent, _ := node["parent_attempt_id"].(string)
					if parent == "" || !known[parent] {
						return errors.New("attempt tree parent must precede child")
					}
				}
				for field, expected := range map[string]string{"task_id": record.TaskID, "operation": c.ID, "provenance": record.Provenance} {
					if value, _ := node[field].(string); value != expected {
						return fmt.Errorf("attempt tree %s cannot replace runtime identity", field)
					}
				}
				if number, ok := node["attempt_number"].(float64); !ok || int(number) != record.AttemptNumber {
					return errors.New("attempt tree ordinal cannot change")
				}
				outcome, _ := node["outcome"].(string)
				if outcome == "verified_success" {
					evidence, _ := node["evidence_refs"].([]any)
					if len(evidence) == 0 || len(evidence) > 16 {
						return errors.New("verified attempt tree node requires evidence")
					}
				}
				if outcome == "failed" {
					fingerprint, _ := node["failure_fingerprint"].(string)
					if fingerprint == "" {
						return errors.New("failed attempt tree node requires a fingerprint")
					}
				}
				known[id] = true
			}
		}
		return nil
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

func sameLearningCohort(a, b string) bool {
	return a == b || ((a == "agent" || a == "operator") && (b == "agent" || b == "operator"))
}
