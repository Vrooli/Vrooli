// Package wizard mounts the WizardService over the deterministic
// interview engine (internal/wizard): session lifecycle, answer
// recording, dry-run scaffold previews, and the explicit apply.
package wizard

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"business-health/internal/checks"
	"business-health/internal/module"
	wizardengine "business-health/internal/wizard"

	wizardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/wizard"
	wizardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/wizard/wizard_v1connect"
)

var ProtoFile = wizardv1.File_business_health_v1_wizard_wizard_proto

// Validator re-runs the contract checks after apply (the round-trip
// guarantee surfaces as residual_findings — expected empty).
type Validator interface {
	ValidateScenario(ctx context.Context, scenario, path string) (checks.Report, error)
}

type handler struct {
	logger    *log.Logger
	engine    *wizardengine.Engine
	validator Validator
	repoRoot  string
}

// Module wires the wizard service. dataDir is business-health's own data
// directory (session persistence); validator re-checks applied scaffolds;
// hinter is the capability-dedup hook (nil = no hints).
func Module(logger *log.Logger, repoRoot, dataDir string, validator Validator, hinter wizardengine.Hinter) module.Module {
	h := &handler{
		logger:    logger,
		engine:    wizardengine.NewEngine(dataDir, hinter, nil),
		validator: validator,
		repoRoot:  repoRoot,
	}
	path, svc := wizardconnect.NewWizardServiceHandler(h)
	return module.Module{
		Name: "wizard",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

func (h *handler) targetDir(scenario, path string) (string, error) {
	if path != "" {
		return path, nil
	}
	if scenario == "" {
		return "", fmt.Errorf("scenario is required")
	}
	dir := filepath.Join(h.repoRoot, "scenarios", scenario)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("scenario %q not found: %w", scenario, err)
	}
	return dir, nil
}

func (h *handler) StartSession(ctx context.Context, req *connect.Request[wizardv1.StartSessionRequest]) (*connect.Response[wizardv1.SessionState], error) {
	dir, err := h.targetDir(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s, err := h.engine.StartSession(req.Msg.GetScenario(), dir, req.Msg.GetReset_())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(sessionState(s, h.engine)), nil
}

func (h *handler) SubmitAnswers(ctx context.Context, req *connect.Request[wizardv1.SubmitAnswersRequest]) (*connect.Response[wizardv1.SessionState], error) {
	s, err := h.engine.LoadSessionByID(req.Msg.GetSessionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	answers := make([]wizardengine.Answer, 0, len(req.Msg.GetAnswers()))
	for _, a := range req.Msg.GetAnswers() {
		answers = append(answers, answerFromProto(a))
	}
	invalid, err := h.engine.SubmitAnswers(&s, answers)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	state := sessionState(s, h.engine)
	for id, reason := range invalid {
		state.Answers[id] = &wizardv1.Answer{QuestionId: id, InvalidReason: reason}
	}
	return connect.NewResponse(state), nil
}

func (h *handler) PreviewScaffold(ctx context.Context, req *connect.Request[wizardv1.PreviewScaffoldRequest]) (*connect.Response[wizardv1.ScaffoldPreview], error) {
	s, err := h.engine.LoadSessionByID(req.Msg.GetSessionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	files, blocking, err := h.engine.Scaffold(s)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &wizardv1.ScaffoldPreview{SessionId: s.ID, Blocking: blocking}
	for _, f := range files {
		out.Files = append(out.Files, &wizardv1.FileDiff{Path: f.Path, Before: f.Before, After: f.After})
	}
	return connect.NewResponse(out), nil
}

func (h *handler) ApplyScaffold(ctx context.Context, req *connect.Request[wizardv1.ApplyScaffoldRequest]) (*connect.Response[wizardv1.ScaffoldResult], error) {
	if !req.Msg.GetApply() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("apply=true is required (dry-run-by-default: use PreviewScaffold for diffs)"))
	}
	s, err := h.engine.LoadSessionByID(req.Msg.GetSessionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	written, err := h.engine.Apply(s)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	result := &wizardv1.ScaffoldResult{SessionId: s.ID, Written: written}
	if h.validator != nil {
		report, err := h.validator.ValidateScenario(ctx, s.Scenario, s.TargetDir)
		if err != nil {
			result.ResidualFindings = []string{fmt.Sprintf("post-apply validation errored: %v", err)}
		} else {
			for _, f := range report.Findings {
				result.ResidualFindings = append(result.ResidualFindings, fmt.Sprintf("[%s] %s — %s", f.Severity, f.Code, f.Message))
			}
		}
	}
	return connect.NewResponse(result), nil
}

func sessionState(s wizardengine.Session, e *wizardengine.Engine) *wizardv1.SessionState {
	state := &wizardv1.SessionState{
		SessionId: s.ID,
		Scenario:  s.Scenario,
		Answers:   map[string]*wizardv1.Answer{},
		Remaining: wizardengine.Remaining(s),
		Complete:  wizardengine.Complete(s),
	}
	for _, q := range wizardengine.Questions() {
		state.Questions = append(state.Questions, &wizardv1.Question{
			Id:         q.ID,
			Target:     q.Target,
			Prompt:     q.Prompt,
			Help:       q.Help,
			Kind:       q.Kind,
			Required:   q.Required,
			MinEntries: int32(q.MinEntries),
		})
	}
	for id, a := range s.Answers {
		state.Answers[id] = answerToProto(id, a)
	}
	for _, hint := range e.Hints(s) {
		state.Hints = append(state.Hints, &wizardv1.CapabilityHint{
			Scenario:   hint.Scenario,
			Capability: hint.Capability,
			Anchor:     hint.Anchor,
			Score:      float32(hint.Score),
		})
	}
	return state
}

func answerFromProto(a *wizardv1.Answer) wizardengine.Answer {
	out := wizardengine.Answer{
		QuestionID: a.GetQuestionId(),
		Text:       a.GetText(),
		Items:      a.GetItems(),
	}
	for _, t := range a.GetTargets() {
		out.Targets = append(out.Targets, wizardengine.OTAnswer{Title: t.GetTitle(), Description: t.GetDescription()})
	}
	return out
}

func answerToProto(id string, a wizardengine.Answer) *wizardv1.Answer {
	out := &wizardv1.Answer{QuestionId: id, Text: a.Text, Items: a.Items}
	for _, t := range a.Targets {
		out.Targets = append(out.Targets, &wizardv1.OperationalTargetAnswer{Title: t.Title, Description: t.Description})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "wizard_start_session",
		Path:        wizardconnect.WizardServiceStartSessionProcedure,
		Method:      "POST",
		Summary:     "Start or resume a contract-authoring wizard session",
		Description: "Creates (or resumes) the deterministic interview session for scaffolding a conformant PRD.md + requirements/ skeleton. Question model derives from the same template definitions the validator checks.",
		Category:    "wizard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (required)", "path": "string (optional resolved target)", "reset": "boolean (discard the existing session)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"session_id": "string", "questions": "array<Question>", "answers": "map<string, Answer>", "remaining": "array<string>", "complete": "boolean", "hints": "array<CapabilityHint>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing scenario or unresolvable target"}},
		Examples: []module.Example{
			{Name: "Start a session", Curl: "curl http://localhost:${API_PORT}/vrooli.business_health.v1.wizard.WizardService/StartSession -H 'Content-Type: application/json' -d '{\"scenario\":\"my-scenario\"}'"},
		},
	},
	{
		ID:          "wizard_submit_answers",
		Path:        wizardconnect.WizardServiceSubmitAnswersProcedure,
		Method:      "POST",
		Summary:     "Record wizard answers",
		Description: "Validates and records answers into a session (invalid answers come back with reasons; valid ones persist).",
		Category:    "wizard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"session_id": "string", "answers": "array<Answer>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"remaining": "array<string>", "complete": "boolean"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Unknown session id"}},
	},
	{
		ID:          "wizard_preview_scaffold",
		Path:        wizardconnect.WizardServicePreviewScaffoldProcedure,
		Method:      "POST",
		Summary:     "Preview the scaffold as diffs (dry-run)",
		Description: "Renders the PRD.md + requirements/ artifacts the current answers produce, as diffs against the target tree. Never writes.",
		Category:    "wizard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"session_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"files": "array<FileDiff>", "blocking": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Unknown session id"}},
	},
	{
		ID:          "wizard_apply_scaffold",
		Path:        wizardconnect.WizardServiceApplyScaffoldProcedure,
		Method:      "POST",
		Summary:     "Write the previewed scaffold (explicit apply)",
		Description: "Writes the previewed artifacts (requires apply=true and a complete answer set), then re-validates the target — residual_findings is expected empty (the round-trip guarantee).",
		Category:    "wizard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"session_id": "string", "apply": "boolean (required true)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"written": "array<string>", "residual_findings": "array<string>"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "apply=true missing"},
			{Status: 412, Code: "failed_precondition", Description: "Required questions unanswered"},
		},
	},
}
