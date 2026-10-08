// finite_serial_scheduler.go consumes owned codec results without exposing a public child caller.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
	"bytes"
	"context"
	"encoding/json"

	"github.com/vrooli/api-core/effortauthority"
	"io"
	"time"
)

type serialTaskRoot struct {
	TaskID         string `json:"taskID"`
	RootTaskDigest string `json:"rootTaskDigest"`
	RootTag        string `json:"rootTag"`
}
type serialCommand struct {
	Version     int    `json:"version"`
	NextProfile string `json:"next_profile"`
	Title       string `json:"title"`
	Instruction string `json:"instruction"`
	Summary     string `json:"summary"`
}
type serialPayload struct {
	serialTaskRoot
	Command            serialCommand `json:"command"`
	SourceResultDigest string        `json:"sourceResultDigest"`
}

func decodeSerial(data []byte, out any) error {
	if len(data) == 0 || len(data) > 24576 {
		return effortauthority.ErrRefused
	}
	if !serialUnambiguousJSON(data) {
		return effortauthority.ErrRefused
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return effortauthority.ErrRefused
	}
	return nil
}

// Exact spelling and one occurrence per object are required; encoding/json's
// compatibility aliases must not choose between competing episode commands.
func serialUnambiguousJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	allowed := map[string]bool{"finite_serial": true, "version": true, "next_profile": true, "title": true, "instruction": true, "summary": true, "taskID": true, "rootTaskDigest": true, "rootTag": true, "command": true, "sourceResultDigest": true}
	var value func() bool
	value = func() bool {
		tok, e := d.Token()
		if e != nil {
			return false
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return true
		}
		if delim != '{' {
			return false
		}
		seen := map[string]bool{}
		for d.More() {
			key, e := d.Token()
			k, ok := key.(string)
			if e != nil || !ok || !allowed[k] || seen[k] {
				return false
			}
			seen[k] = true
			if !value() {
				return false
			}
		}
		close, e := d.Token()
		return e == nil && close == json.Delim('}')
	}
	if !value() {
		return false
	}
	_, e := d.Token()
	return e == io.EOF
}

func validateSerialTaskProjection(t *domain.Task, payload []byte) error {
	var p serialPayload
	if t == nil || decodeSerial(payload, &p) != nil || t.ID.String() != p.TaskID || effortTaskDigest(t) != p.RootTaskDigest {
		return effortauthority.ErrRefused
	}
	return nil
}
func serialProjectedTask(t *domain.Task, p serialPayload) *domain.Task {
	c := *t
	c.Description = t.Description + "\n\nFinite episode action (retain the complete original task and required sources):\n" + p.Command.Title + "\n" + p.Command.Instruction + "\nPrevious episode handover:\n" + p.Command.Summary
	return &c
}
func serialSelectedCommand(r *domain.Run) (serialCommand, error) {
	var out struct {
		Command serialCommand `json:"finite_serial"`
	}
	if r == nil || !r.Status.IsTerminal() || r.Result == nil || !r.Result.Success || r.Result.ExitCode != 0 || r.Result.Selection.Status != domain.FinalOutputSelectionSelected || r.Result.Selection.SelectedCandidateID == "" {
		return serialCommand{}, effortauthority.ErrRefused
	}
	matches := 0
	for _, c := range r.Result.Candidates {
		if c.ID == r.Result.Selection.SelectedCandidateID {
			if c.Content != r.Result.FinalOutput {
				return serialCommand{}, effortauthority.ErrRefused
			}
			matches++
		}
	}
	if matches != 1 || decodeSerial([]byte(r.Result.FinalOutput), &out) != nil {
		return serialCommand{}, effortauthority.ErrRefused
	}
	var envelope map[string]json.RawMessage
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(r.Result.FinalOutput), &envelope) != nil || json.Unmarshal(envelope["finite_serial"], &fields) != nil || len(envelope) != 1 || len(fields) != 5 {
		return serialCommand{}, effortauthority.ErrRefused
	}
	c := out.Command
	if c.Version != 1 || c.NextProfile == "" || len(c.NextProfile) > 256 || c.Title == "" || len(c.Title) > 200 || c.Instruction == "" || len(c.Instruction) > 4096 || len(c.Summary) > 4096 {
		return serialCommand{}, effortauthority.ErrRefused
	}
	return c, nil
}
func (o *Orchestrator) observeFiniteSerialTerminal(r *domain.Run) {
	o.projectTerminalInvocationReadModel(r)
	// No public endpoint reaches this adapter. Owner reconciliation repeats only
	// the same deterministic key; uncertainty never supplies another reservation.
	if err := o.advanceFiniteSerialEpisode(context.Background(), r); err != nil {
		obs.Component("finite-serial").Warn("finite episode held for owner reconciliation", obs.KeyRunID, r.ID.String(), obs.KeyError, err.Error())
	}
}
func (o *Orchestrator) advanceFiniteSerialEpisode(ctx context.Context, r *domain.Run) error {
	if r == nil || r.ResolvedConfig == nil || r.ResolvedConfig.Admission == nil || r.ResolvedConfig.Admission.Effort == nil {
		return nil
	}
	if o.effortAuthority == nil {
		return effortauthority.ErrRefused
	}
	a := r.ResolvedConfig.Admission
	b := *a.Effort
	p, e := o.effortAuthority.CheckBinding(ctx, b)
	if e != nil {
		return e
	}
	if len(p.SerialEdges) == 0 {
		return nil
	}
	engine, ok := o.effortAuthority.(effortauthority.SerialPayloadEngine)
	if !ok || a.EffortIntent == nil || !r.Status.IsTerminal() {
		return effortauthority.ErrRefused
	}
	source, e := o.effortAuthority.ReadReservation(ctx, b, r.IdempotencyKey)
	if e != nil || !source.NativeBound || source.RunID != r.ID.String() || source.IntentDigest != effortauthority.Digest(*a.EffortIntent) {
		return effortauthority.ErrRefused
	}
	currentSource, e := o.profiles.GetByKey(ctx, source.Intent.Profile)
	if e != nil || currentSource == nil || EffortProfileDigest(currentSource) != source.Intent.ProfileDigest || p.Profiles[source.Intent.Profile] != source.Intent.ProfileDigest || r.OwnerSubject != b.Owner || r.OwnerExpiresAt == nil || !r.OwnerExpiresAt.Equal(b.Deadline) || !r.OwnerExpiresAt.After(o.now()) {
		return effortauthority.ErrRefused
	}
	// A receipt qualifies this source's recorded execution, not a replacement
	// effort/model supplied by an observed callback or a corrupt retained row.
	// Keep ordinary delegation narrowing unchanged; refuse inconsistent source
	// evidence before writing handoff metadata or settling any reservation.
	receipt := a.Receipt
	if receipt == nil || receipt.EffectiveRunner != string(r.ResolvedConfig.RunnerType) || receipt.EffectiveModel != r.ResolvedConfig.Model || receipt.EffectiveEffort != string(r.ResolvedConfig.Effort) {
		return effortauthority.ErrRefused
	}
	key := "serial-" + r.ID.String()
	h, e := engine.ReadSerialHandoff(ctx, b, key)
	if e != nil {
		// Only a real owner terminal proof can authenticate result-driven metadata.
		if source.Terminal || o.finiteNativeTerminal == nil || o.finiteNativeTerminal.Terminal(ctx, r.ID.String()) != nil {
			return effortauthority.ErrRefused
		}
		c, e := serialSelectedCommand(r)
		if e != nil {
			return e
		}
		var root serialTaskRoot
		var previous serialPayload
		if decodeSerial(a.EffortTaskProjection, &root) != nil {
			if decodeSerial(a.EffortTaskProjection, &previous) != nil {
				return effortauthority.ErrRefused
			}
			root = previous.serialTaskRoot
		}
		task, e := o.GetTask(ctx, r.TaskID)
		if e != nil || task == nil || task.ID.String() != root.TaskID || effortTaskDigest(task) != root.RootTaskDigest {
			return effortauthority.ErrRefused
		}
		sourceTask := task
		sourceInputReq := CreateRunRequest{TaskID: r.TaskID, ParentRunID: r.ParentRunID, IdempotencyKey: r.IdempotencyKey, Tag: root.RootTag, Environment: r.CustomEnv}
		if source.Intent.Effect == "run.child" {
			if effortauthority.Digest(string(a.EffortTaskProjection)) != source.Intent.SerialPayloadDigest {
				return effortauthority.ErrRefused
			}
			sourceTask = serialProjectedTask(task, previous)
			sourceInputReq.Tag = r.Tag
			sourceInputReq.effort = &effortAdmission{intent: source.Intent}
		}
		if effortauthority.Digest(effortNativeInput(sourceInputReq, sourceTask, source.Intent.Profile)) != source.Intent.InputDigest {
			return effortauthority.ErrRefused
		}
		next := source.Intent
		next.Effect = "run.child"
		next.Endpoint = "/api/v1/runs"
		next.Method = "POST"
		next.ParentRunID = r.ID.String()
		next.SourceRunID = ""
		next.IdempotencyKey = key
		found := false
		for _, edge := range p.SerialEdges {
			if edge.FromMember == source.Intent.Member && edge.FromProfile == source.Intent.Profile && edge.ToProfile == c.NextProfile {
				next.Member = edge.ToMember
				next.Profile = edge.ToProfile
				found = true
			}
		}
		if !found {
			return effortauthority.ErrRefused
		}
		profile, e := o.profiles.GetByKey(ctx, next.Profile)
		if e != nil || profile == nil || EffortProfileDigest(profile) != p.Profiles[next.Profile] {
			return effortauthority.ErrRefused
		}
		next.ProfileDigest = p.Profiles[next.Profile]
		next.Turns = p.MaxTurns
		next.ToolCalls = p.MaxToolCalls
		next.RunSeconds = p.MaxRunSeconds
		payload, e := json.Marshal(serialPayload{serialTaskRoot: root, Command: c, SourceResultDigest: effortauthority.Digest(r.Result)})
		if e != nil || len(payload) > 24576 {
			return effortauthority.ErrRefused
		}
		next.SerialPayloadDigest = effortauthority.Digest(string(payload))
		req := CreateRunRequest{TaskID: r.TaskID, ParentRunID: &r.ID, IdempotencyKey: key, ProfileRef: &ProfileRef{ProfileKey: next.Profile}, Environment: r.CustomEnv, Tag: r.Tag, effort: &effortAdmission{intent: next}}
		req.profileAdmission = &profileAdmissionPlan{profile: profile}
		cfg, _, e := o.resolveRunConfig(ctx, req)
		if e != nil || cfg == nil {
			return effortauthority.ErrRefused
		}
		cfg.SandboxConfig, e = o.resolveSandboxConfig(req, profile)
		if e != nil {
			return e
		}
		if e = o.validateCurrentExecutionModel(ctx, cfg); e != nil {
			return e
		}
		// The existing exact-parent receipt is required before metadata too; the
		// complete ordinary admission gate is repeated before reservation below.
		if e = o.admitDependentDelegation(ctx, req, cfg); e != nil {
			return e
		}
		projected := serialProjectedTask(task, serialPayload{serialTaskRoot: root, Command: c})
		next.InputDigest = effortauthority.Digest(effortNativeInput(req, projected, next.Profile))
		h, _, e = engine.PrepareSerialHandoffWithPayload(ctx, b, r.IdempotencyKey, next, payload)
		if e != nil {
			return e
		}
	}
	if !source.Terminal {
		if e = o.SyncEffortTerminal(ctx, b, r.IdempotencyKey, r.ID.String()); e != nil {
			return e
		}
	}
	return o.createFiniteSerialSuccessor(ctx, r, b, p, h)
}
func (o *Orchestrator) revalidateSerialCaller(ctx context.Context, req CreateRunRequest) error {
	if req.effort == nil || !req.effort.serial || req.caller == nil || req.caller.Kind != "effort-serial-native-owner" || req.caller.Subject != req.effort.binding.Owner || req.ParentRunID == nil || req.caller.RunID != *req.ParentRunID || req.OwnerSubject != req.effort.binding.Owner || req.OwnerExpiresAt == nil || !req.OwnerExpiresAt.Equal(req.effort.binding.Deadline) {
		return effortauthority.ErrRefused
	}
	parent, e := o.runs.Get(ctx, *req.ParentRunID)
	if e != nil || parent == nil || parent.OwnerSubject != req.OwnerSubject || !sameCreateRunScopeCeiling(parent.OwnerScopes, req.OwnerScopes) || parent.OwnerExpiresAt == nil || !parent.OwnerExpiresAt.Equal(*req.OwnerExpiresAt) {
		return effortauthority.ErrRefused
	}
	engine, ok := o.effortAuthority.(effortauthority.SerialPayloadEngine)
	if !ok {
		return effortauthority.ErrRefused
	}
	h, e := engine.ReadSerialHandoff(ctx, req.effort.binding, req.IdempotencyKey)
	if e != nil || h.SourceRunID != req.ParentRunID.String() || req.IdempotencyKey != "serial-"+h.SourceRunID || effortauthority.Digest(h.Next) != effortauthority.Digest(req.effort.intent) || !bytes.Equal(h.Payload, req.effort.serialPayload) {
		return effortauthority.ErrRefused
	}
	source, e := o.effortAuthority.ReadReservation(ctx, req.effort.binding, h.SourceKey)
	if e != nil || !source.Terminal || !source.NativeBound || source.RunID != h.SourceRunID {
		return effortauthority.ErrRefused
	}
	return nil
}
func (o *Orchestrator) createFiniteSerialSuccessor(ctx context.Context, source *domain.Run, b effortauthority.Binding, p effortauthority.Policy, h effortauthority.SerialHandoff) error {
	var payload serialPayload
	if decodeSerial(h.Payload, &payload) != nil || h.SourceRunID != source.ID.String() || payload.TaskID != source.TaskID.String() || payload.SourceResultDigest != effortauthority.Digest(source.Result) || h.Next.SerialPayloadDigest != effortauthority.Digest(string(h.Payload)) {
		return effortauthority.ErrRefused
	}
	task, e := o.GetTask(ctx, source.TaskID)
	if e != nil || validateSerialTaskProjection(task, h.Payload) != nil {
		return effortauthority.ErrRefused
	}
	req := CreateRunRequest{TaskID: source.TaskID, ParentRunID: &source.ID, IdempotencyKey: h.Next.IdempotencyKey, ProfileRef: &ProfileRef{ProfileKey: h.Next.Profile}, Environment: source.CustomEnv, Tag: source.Tag,
		OwnerSubject: b.Owner, OwnerScopes: append([]string(nil), source.OwnerScopes...), OwnerExpiresAt: &b.Deadline,
		caller: &domain.CreateRunCaller{Kind: "effort-serial-native-owner", Subject: b.Owner, RunID: source.ID},
		effort: &effortAdmission{binding: b, policy: p, intent: h.Next, serial: true, serialPayload: append([]byte(nil), h.Payload...), serialTask: serialProjectedTask(task, payload)}}
	bindEffortCaller(&req)
	if e = o.revalidateSerialCaller(ctx, req); e != nil {
		return e
	}
	_, e = o.createRun(ctx, req, nil)
	return e
}

// The normal owner reconcile loop reads a bounded recent terminal window. It
// performs no API-triggered refresh and never invents a key after uncertainty.
func (o *Orchestrator) ReconcileFiniteSerialEpisodes(ctx context.Context) error {
	if o.effortAuthority == nil || !o.finiteNativeEnabled() {
		return nil
	}
	o.serialScanMu.Lock()
	defer o.serialScanMu.Unlock()
	if o.serialScanThrough.IsZero() {
		o.serialScanThrough = o.now().UTC()
		o.serialScanFrom = o.serialScanThrough.Add(-8 * time.Hour)
		o.serialScanOffset = 0
	}
	status := domain.RunStatusComplete
	runs, e := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 256, Offset: o.serialScanOffset}, Status: &status, EndedFrom: &o.serialScanFrom, EndedTo: &o.serialScanThrough})
	if e != nil {
		return e
	}
	var first error
	for _, r := range runs {
		if e := o.advanceFiniteSerialEpisode(ctx, r); e != nil && first == nil {
			first = e
		}
	}
	if len(runs) == 256 {
		o.serialScanOffset += len(runs)
	} else {
		o.serialScanThrough = time.Time{}
		o.serialScanOffset = 0
	}
	// The fixed EndedTo excludes newly completed rows until the next sweep. A
	// restart begins a fresh sweep; no cursor is authority or a dispatch receipt.
	return first
}
