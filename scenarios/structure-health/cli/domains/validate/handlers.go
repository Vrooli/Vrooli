package validate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: validationconnect.NewValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg

	results := make([]string, 0, len(msg.GetFindings()))
	for _, f := range msg.GetFindings() {
		fix := ""
		if f.GetAutofixAvailable() {
			fix = " (auto-fixable)"
		}
		results = append(results, fmt.Sprintf("[%s] %s %s: %s%s", f.GetSeverity(), f.GetCode(), f.GetLocation(), f.GetMessage(), fix))
	}
	if len(results) == 0 {
		results = append(results, "No structure findings.")
	}

	summary := []string{
		fmt.Sprintf("%s (%s): %s", msg.GetScenario(), msg.GetStatus(), msg.GetSummary()),
	}
	if p := msg.GetProfile(); p != nil {
		recognized := "recognized"
		if !p.GetRecognized() {
			recognized = "unrecognized → advisory rules"
		}
		summary = append(summary, fmt.Sprintf("Profile: %s [%s]", p.GetId(), recognized))
	}
	if reason := strings.TrimSpace(msg.GetDegradedReason()); reason != "" {
		summary = append(summary, "Degraded: "+reason)
	}
	summary = append(summary, surfaceLines(msg.GetSurfaces())...)

	human := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: msg.GetNextSteps(),
	}
	if maturity := maturityreport.BuildMaturityListReport(msg.GetAssessment()); maturity.Summary != nil {
		human.Summary = append(human.Summary, maturity.Summary...)
		human.RetrievalHints = append(human.RetrievalHints, maturity.RetrievalHints...)
	}
	if err := cliapp.RenderProtoList(ctx, msg, human); err != nil {
		return err
	}
	if strings.EqualFold(msg.GetStatus(), "failed") {
		return fmt.Errorf("structure-health validation failed for %q", scenario)
	}
	return nil
}

func (h *handlers) validateTarget(kind string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		path := firstFlag(ctx.FlagValues("path"))
		id := strings.TrimSpace(ctx.Positional("target"))
		if id == "" {
			if kind == "project" {
				id = "repo"
			} else {
				id = kind
			}
		}
		if strings.TrimSpace(path) == "" && kind != "project" {
			return fmt.Errorf("validate %s requires --path so the target root is unambiguous", kind)
		}
		if path == "" {
			path = "."
		}
		resp, err := h.validateTargetResponse(ctx, kind, id, path)
		if err != nil {
			return cliapp.WrapAPIError(fmt.Sprintf("validate %s %q", kind, id), err, nil)
		}
		return renderTarget(ctx, kind, id, resp)
	}
}

func (h *handlers) validateTargetResponse(ctx cliapp.RunContext, kind, id, path string) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(ctx.Core())
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
	resp, err := client.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{Kind: targetKind(kind), Id: id, Root: path}, Path: path,
	}))
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no validation response")
	}
	return resp, nil
}

func renderTarget(ctx cliapp.RunContext, kind, id string, resp *connect.Response[scenariovalidationv1.ValidateTargetResponse]) error {
	results := make([]string, 0)
	if assessment := resp.Msg.GetAssessment(); assessment != nil {
		for _, f := range assessment.GetFindings() {
			if f == nil {
				continue
			}
			results = append(results, fmt.Sprintf("[%s] %s %s: %s — %s", f.GetSeverity(), f.GetCode(), f.GetLocation(), f.GetMessage(), f.GetRemediation()))
		}
	}
	if len(results) == 0 {
		results = append(results, "No structure findings.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%s/%s (%s)", kind, id, resp.Msg.GetStatus())}, ResultsHeading: "Findings", Results: results})
}

func (h *handlers) validateAll(ctx cliapp.RunContext) error {
	root := firstFlag(ctx.FlagValues("path"))
	if root == "" {
		root, _ = os.Getwd()
	}
	root, _ = filepath.Abs(root)
	targets := []struct{ kind, id, root string }{}
	addDirs := func(kind, base string) {
		entries, _ := os.ReadDir(filepath.Join(root, base))
		for _, entry := range entries {
			if entry.IsDir() {
				targets = append(targets, struct{ kind, id, root string }{kind, entry.Name(), filepath.Join(root, base, entry.Name())})
			}
		}
	}
	addDirs("scenario", "scenarios")
	addDirs("resource", "resources")
	addDirs("tool", filepath.Join("internal", "tools"))
	addDirs("safeguard", filepath.Join("internal", "safeguards"))
	addDirs("team", "docs")
	addDirs("package", "packages")
	targets = append(targets, struct{ kind, id, root string }{"control-plane", "cmd", filepath.Join(root, "cmd")}, struct{ kind, id, root string }{"control-plane", "internal", filepath.Join(root, "internal")}, struct{ kind, id, root string }{"docs", "docs", filepath.Join(root, "docs")}, struct{ kind, id, root string }{"project", "repo", root})
	for _, target := range targets {
		resp, err := h.validateTargetResponse(ctx, target.kind, target.id, target.root)
		if err != nil {
			return cliapp.WrapAPIError(fmt.Sprintf("validate %s %q", target.kind, target.id), err, nil)
		}
		if err := renderTarget(ctx, target.kind, target.id, resp); err != nil {
			return err
		}
	}
	return nil
}

func targetKind(kind string) commonv1.ValidationTargetKind {
	values := map[string]commonv1.ValidationTargetKind{
		"scenario":      commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO,
		"resource":      commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE,
		"tool":          commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL,
		"safeguard":     commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD,
		"team":          commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM,
		"package":       commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE,
		"control-plane": commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE,
		"docs":          commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS,
		"project":       commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT,
	}
	return values[kind]
}

// surfaceLines renders the declared-vs-actual reconcile per surface.
func surfaceLines(surfaces []*validationv1.SurfaceReconcile) []string {
	if len(surfaces) == 0 {
		return nil
	}
	lines := []string{fmt.Sprintf("Surfaces (%d):", len(surfaces))}
	for _, s := range surfaces {
		state := "declared+actual"
		switch {
		case s.GetDeclared() && !s.GetActual():
			state = "declared, NOT detected"
		case !s.GetDeclared() && s.GetActual():
			state = "detected, NOT declared"
		}
		lines = append(lines, fmt.Sprintf("  • %s [%s] — %s", s.GetSurface(), s.GetKind(), state))
	}
	return lines
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
