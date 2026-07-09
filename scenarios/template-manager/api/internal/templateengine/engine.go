package templateengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
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
	_, templates, err := runTemplateList(e.deps(), ctx, templatecontracts.TemplateListRequest{})
	return templates, err
}

func (e *Engine) ShowTemplate(ctx context.Context, name string) (templatecontracts.TemplateInfo, error) {
	_, info, err := runTemplateShow(e.deps(), ctx, templatecontracts.TemplateShowRequest{Name: name})
	return info, err
}

func (e *Engine) GenerateScenario(ctx context.Context, req templatecontracts.GenerateRequest) (templatecontracts.GenerateResult, error) {
	_, result, err := runGenerate(e.deps(), ctx, req)
	return result, err
}

func (e *Engine) OrientScenario(ctx context.Context, req templatecontracts.OrientationRequest) (templatecontracts.OrientationReport, error) {
	_, report, err := runOrientation(e.deps(), ctx, req)
	return report, err
}

func (e *Engine) DetemplateScenario(ctx context.Context, req templatecontracts.DetemplateRequest) (templatecontracts.DetemplateResult, error) {
	_, result, err := runDetemplate(e.deps(), ctx, req)
	return result, err
}

func (e *Engine) ValidateTemplate(ctx context.Context, req templatecontracts.TemplateValidateRequest) (templatecontracts.TemplateValidationReport, error) {
	_, report, err := runTemplateValidate(e.deps(), ctx, req)
	return report, err
}

func (e *Engine) DriftReport(ctx context.Context, req templatecontracts.TemplateDriftRequest) (templatecontracts.TemplateDriftReport, error) {
	_, report, err := runTemplateDrift(e.deps(), ctx, req)
	return report, err
}

func (e *Engine) CleanupRuns(ctx context.Context, req templatecontracts.TemplateCleanupRequest) (templatecontracts.TemplateCleanupResult, error) {
	_, result, err := runTemplateCleanup(e.deps(), ctx, req)
	return result, err
}

func (e *Engine) ListDesignKits(ctx context.Context) ([]templatecontracts.DesignKitInfo, error) {
	_, kits, err := runDesignList(e.deps(), ctx, templatecontracts.DesignListRequest{})
	return kits, err
}

func (e *Engine) ShowDesignKit(ctx context.Context, id string) (templatecontracts.DesignKitInfo, error) {
	_, kit, err := runDesignShow(e.deps(), ctx, templatecontracts.DesignShowRequest{ID: id})
	return kit, err
}

func (e *Engine) ValidateDesignKits(ctx context.Context, req templatecontracts.DesignValidateRequest) (templatecontracts.DesignValidationReport, error) {
	_, report, err := runDesignValidate(e.deps(), ctx, req)
	return report, err
}

func (e *Engine) deps() HandlerDeps[context.Context] {
	return HandlerDeps[context.Context]{
		Stdout:       func(context.Context) io.Writer { return io.Discard },
		Stderr:       func(context.Context) io.Writer { return io.Discard },
		Root:         func(context.Context) string { return e.root },
		Globals:      func(context.Context) rootcli.GlobalOptions { return rootcli.GlobalOptions{JSON: true} },
		OutputFormat: func(context.Context) (cliout.Format, error) { return cliout.FormatJSON, nil },
		HomeDir:      func(context.Context) (string, error) { return os.UserHomeDir() },
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
