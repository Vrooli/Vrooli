package validate

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 60*time.Second)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: name}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	for _, finding := range assessment.GetFindings() {
		results = append(results, fmt.Sprintf("[%s] %s: %s", finding.GetSeverity(), finding.GetCode(), finding.GetMessage()))
	}
	summary := []string{fmt.Sprintf("Validated %s — status=%s findings=%d", msg.GetScenario(), msg.GetStatus(), len(results))}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{Summary: summary, ResultsHeading: "Findings", Results: results}); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED || msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		return fmt.Errorf("scenario %q failed storage validation", name)
	}
	return nil
}

func (h *handlers) proveIsolation(ctx cliapp.RunContext) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	scenario := ctx.Positional("name")
	apiDir := filepath.Join(root, "scenarios", scenario, "api")
	required := []struct {
		name   string
		marker string
	}{
		{"database.Open", "database.Open"},
		{"database.EnsureSchemas", "database.EnsureSchemas"},
		{"apihttp.TestModeMiddleware", "apihttp.TestModeMiddleware"},
		{"devrouting.RegisterWithFileRoots", "devrouting.RegisterWithFileRoots"},
		{"filerouting.New", "filerouting.New"},
		{"RoutedRoots.Pick", "RoutedRoots.Pick"},
	}
	var source strings.Builder
	if walkErr := filepath.WalkDir(apiDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		source.Write(data)
		return nil
	}); walkErr != nil {
		return fmt.Errorf("prove isolation: scan %s: %w", scenario, walkErr)
	}
	missing := make([]string, 0)
	for _, check := range required {
		if !strings.Contains(source.String(), check.marker) {
			missing = append(missing, check.name)
		}
	}
	results := []string{fmt.Sprintf("Scenario %s isolation proof: %t", scenario, len(missing) == 0)}
	if len(missing) > 0 {
		results = append(results, "Missing seams: "+strings.Join(missing, ", "))
	}
	if err := ctx.RenderList(cliapp.ListReport{Summary: []string{"API-free static isolation proof"}, ResultsHeading: "Isolation", Results: results}); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("scenario %q isolation is not proven", scenario)
	}
	return nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no repository root found above %s", dir)
		}
		dir = parent
	}
}
