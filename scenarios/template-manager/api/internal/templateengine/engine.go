package templateengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"

	repocontract "github.com/vrooli/repo-contract-go"
)

type Engine struct {
	root string
}

func New(root string) (*Engine, error) {
	if root == "" {
		resolved, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			return nil, fmt.Errorf("resolve repo root: %w", err)
		}
		root = resolved
	}
	return &Engine{root: root}, nil
}

func MustNew(root string) *Engine {
	engine, err := New(root)
	if err != nil {
		panic(err)
	}
	return engine
}

func (e *Engine) ListTemplates(ctx context.Context) ([]templatecontracts.TemplateInfo, error) {
	return runTemplateList(e.deps(), ctx, templatecontracts.TemplateListRequest{})
}

func (e *Engine) ShowTemplate(ctx context.Context, name string) (templatecontracts.TemplateInfo, error) {
	return runTemplateShow(e.deps(), ctx, templatecontracts.TemplateShowRequest{Name: name})
}

func (e *Engine) GenerateScenario(ctx context.Context, req templatecontracts.GenerateRequest) (templatecontracts.GenerateResult, error) {
	return runGenerate(e.deps(), ctx, req)
}

func (e *Engine) OrientScenario(ctx context.Context, req templatecontracts.OrientationRequest) (templatecontracts.OrientationReport, error) {
	return runOrientation(e.deps(), ctx, req)
}

func (e *Engine) DetemplateScenario(ctx context.Context, req templatecontracts.DetemplateRequest) (templatecontracts.DetemplateResult, error) {
	return runDetemplate(e.deps(), ctx, req)
}

func (e *Engine) ValidateTemplate(ctx context.Context, req templatecontracts.TemplateValidateRequest) (templatecontracts.TemplateValidationReport, error) {
	return runTemplateValidate(e.deps(), ctx, req)
}

func (e *Engine) DriftReport(ctx context.Context, req templatecontracts.TemplateDriftRequest) (templatecontracts.TemplateDriftReport, error) {
	return runTemplateDrift(e.deps(), ctx, req)
}

func (e *Engine) CleanupRuns(ctx context.Context, req templatecontracts.TemplateCleanupRequest) (templatecontracts.TemplateCleanupResult, error) {
	return runTemplateCleanup(e.deps(), ctx, req)
}

func (e *Engine) ListDesignKits(ctx context.Context) ([]templatecontracts.DesignKitInfo, error) {
	return runDesignList(e.deps(), ctx, templatecontracts.DesignListRequest{})
}

func (e *Engine) ShowDesignKit(ctx context.Context, id string) (templatecontracts.DesignKitInfo, error) {
	return runDesignShow(e.deps(), ctx, templatecontracts.DesignShowRequest{ID: id})
}

func (e *Engine) ValidateDesignKits(ctx context.Context, req templatecontracts.DesignValidateRequest) (templatecontracts.DesignValidationReport, error) {
	return runDesignValidate(e.deps(), ctx, req)
}

func (e *Engine) deps() HandlerDeps[context.Context] {
	return HandlerDeps[context.Context]{
		Stdout: func(context.Context) io.Writer { return io.Discard },
		Stderr: func(context.Context) io.Writer { return io.Discard },
		Root:   func(context.Context) string { return e.root },
		RunSubprocess: func(_ context.Context, spec scenarioexec.SubprocessSpec) error {
			if spec.Stdout == nil {
				spec.Stdout = &bytes.Buffer{}
			}
			if spec.Stderr == nil {
				spec.Stderr = &bytes.Buffer{}
			}
			return scenarioexec.RunSubprocess(spec)
		},
		LocateTestGenieCLI: func(context.Context) (string, error) {
			home, _ := os.UserHomeDir()
			return scenarioexec.LocateTestGenieCLI(nil, e.root, home)
		},
		CommandEnv: func(context.Context) []string { return os.Environ() },
	}
}
