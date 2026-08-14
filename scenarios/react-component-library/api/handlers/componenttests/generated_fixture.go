package componenttests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
)

const (
	generatedFixtureLibraryID   = "react-component-library:Button"
	generatedFixtureAdoptedPath = "ui/src/components/ui/button.tsx"
	generatedFixtureTemplate    = "react-vite"
)

// GeneratedFixtureValidator is the narrow seam used by the Test Genie
// provider. Keeping it injectable lets the provider remain unit-testable
// without making a test depend on Template Manager, a browser, or the host
// control plane.
type GeneratedFixtureValidator interface {
	Validate(context.Context) error
}

type generatedFixtureValidator struct {
	adoptions  adoptions.Service
	components components.Service
	sourceRoot string
	logger     *log.Logger
}

func NewGeneratedFixtureValidator(adoptionService adoptions.Service, componentService components.Service, sourceRoot string, logger *log.Logger) GeneratedFixtureValidator {
	return &generatedFixtureValidator{adoptions: adoptionService, components: componentService, sourceRoot: sourceRoot, logger: logger}
}

func (v *generatedFixtureValidator) Validate(ctx context.Context) error {
	root, err := repositoryRoot(v.sourceRoot)
	if err != nil {
		return err
	}
	component, err := v.components.GetByLibraryID(ctx, generatedFixtureLibraryID)
	if err != nil {
		return fmt.Errorf("resolve generated-fixture component: %w", err)
	}
	version := strings.TrimSpace(component.LatestVersion)
	if version == "" {
		return fmt.Errorf("generated-fixture component %s has no latest version", generatedFixtureLibraryID)
	}

	positive := fixtureName("positive")
	negative := fixtureName("negative")
	var positiveAdoptionID string
	defer func() {
		v.cleanup(context.Background(), root, positive, positiveAdoptionID)
		v.cleanup(context.Background(), root, negative, "")
	}()

	if err := v.generate(ctx, root, positive, "RCL Generated Suite Positive"); err != nil {
		return err
	}
	positivePreflight, err := v.adoptions.Preflight(ctx, adoptions.PreflightInput{ComponentID: component.ID, Scenario: positive, Version: version})
	if err != nil {
		return fmt.Errorf("positive generated-fixture preflight: %w", err)
	}
	if len(positivePreflight.Tokens.Unsatisfied) != 0 {
		return fmt.Errorf("positive generated-fixture preflight has unsatisfied tokens: %s", strings.Join(positivePreflight.Tokens.Unsatisfied, ", "))
	}
	applyResult, err := v.adoptions.Apply(ctx, adoptions.ApplyInput{
		ComponentID:        component.ID,
		Scenario:           positive,
		AdoptedPath:        generatedFixtureAdoptedPath,
		Version:            version,
		ConfirmOverwrite:   true,
		OverrideValidation: true,
		ReplaceExisting:    true,
	})
	if err != nil {
		return fmt.Errorf("apply Button to positive generated fixture: %w", err)
	}
	positiveAdoptionID = applyResult.Adoption.ID

	if err := runControlPlane(ctx, root, "vrooli", "scenario", "start", positive, "--timeout", "240", "--json"); err != nil {
		return fmt.Errorf("start positive generated fixture: %w", err)
	}
	port, err := scenarioUIPort(ctx, root, positive)
	if err != nil {
		return err
	}
	if err := runGeneratedFixtureBrowser(ctx, root, port); err != nil {
		return err
	}

	if err := v.generate(ctx, root, negative, "RCL Generated Suite Negative"); err != nil {
		return err
	}
	if err := removeRequiredSurfaceToken(filepath.Join(root, "scenarios", negative, "ui", "src", "design-tokens.css")); err != nil {
		return fmt.Errorf("remove required token from negative generated fixture: %w", err)
	}
	negativePreflight, err := v.adoptions.Preflight(ctx, adoptions.PreflightInput{ComponentID: component.ID, Scenario: negative, Version: version})
	if err != nil {
		return fmt.Errorf("negative generated-fixture preflight: %w", err)
	}
	if !contains(negativePreflight.Tokens.Unsatisfied, "--color-surface") {
		return fmt.Errorf("negative generated-fixture preflight did not report --color-surface: %v", negativePreflight.Tokens.Unsatisfied)
	}
	return nil
}

func (v *generatedFixtureValidator) generate(ctx context.Context, root, name, displayName string) error {
	return runControlPlane(ctx, root, "template-manager", "lifecycle", "generate", generatedFixtureTemplate,
		"--id", name,
		"--display-name", displayName,
		"--description", "Temporary Test Genie fixture for the RCL adoption contract",
		"--dest", filepath.Join("scenarios", name),
		"--design", "vrooli-default",
		"--json")
}

func (v *generatedFixtureValidator) cleanup(parent context.Context, root, name, adoptionID string) {
	if strings.TrimSpace(name) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	if adoptionID != "" && v.adoptions != nil {
		if _, err := v.adoptions.DeleteWithOptions(ctx, adoptionID, true); err != nil {
			v.logf("generated fixture cleanup adoption %s: %v", adoptionID, err)
		}
	}
	_ = runControlPlane(ctx, root, "vrooli", "scenario", "stop", name, "--json")
	output, err := runControlPlaneOutput(ctx, root, "template-manager", "lifecycle", "destroy", name, "--force", "--json")
	if err != nil {
		v.logf("generated fixture cleanup destroy %s: %v", name, err)
		return
	}
	var result struct {
		NeedsProtoGenerate bool `json:"needs_proto_generate"`
	}
	if json.Unmarshal(output, &result) == nil && result.NeedsProtoGenerate {
		if err := runControlPlane(ctx, root, "make", "-C", filepath.Join(root, "packages", "proto"), "generate"); err != nil {
			v.logf("generated fixture cleanup proto generation: %v", err)
		}
	}
}

func runGeneratedFixtureBrowser(ctx context.Context, root string, port int) error {
	script := filepath.Join(root, "scenarios", "react-component-library", "ui", "scripts", "generated-fixture-e2e.mjs")
	return runControlPlane(ctx, root, "node", script, fmt.Sprintf("http://127.0.0.1:%d/", port))
}

func scenarioUIPort(ctx context.Context, root, name string) (int, error) {
	output, err := runControlPlaneOutput(ctx, root, "vrooli", "scenario", "port", name, "ui", "--json")
	if err != nil {
		return 0, fmt.Errorf("resolve UI port for generated fixture: %w", err)
	}
	var result struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("decode generated fixture UI port: %w", err)
	}
	if result.Port <= 0 {
		return 0, fmt.Errorf("generated fixture returned invalid UI port %d", result.Port)
	}
	return result.Port, nil
}

func runControlPlane(ctx context.Context, root, name string, args ...string) error {
	_, err := runControlPlaneOutput(ctx, root, name, args...)
	return err
}

func runControlPlaneOutput(ctx context.Context, root, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- names are fixed provider-owned control-plane tools.
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(string(output))
		}
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, detail)
	}
	return output, nil
}

func repositoryRoot(sourceRoot string) (string, error) {
	for _, candidate := range []string{os.Getenv("VROOLI_SOURCE_ROOT"), os.Getenv("VROOLI_ROOT"), sourceRoot} {
		current := filepath.Clean(strings.TrimSpace(candidate))
		for current != "." && current != string(filepath.Separator) {
			if _, err := os.Stat(filepath.Join(current, "templates", "scenarios", "react-vite")); err == nil {
				return current, nil
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	return "", fmt.Errorf("locate repository root for generated fixture")
}

func fixtureName(kind string) string {
	return fmt.Sprintf("rcl-fixture-%s-%d", kind, time.Now().UnixNano())
}

func removeRequiredSurfaceToken(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	lines := strings.SplitAfter(string(content), "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--color-surface:") {
			continue
		}
		kept = append(kept, line)
	}
	return os.WriteFile(path, []byte(strings.Join(kept, "")), mode)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (v *generatedFixtureValidator) logf(format string, args ...any) {
	if v.logger != nil {
		v.logger.Printf(format, args...)
	}
}
