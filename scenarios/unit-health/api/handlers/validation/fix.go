package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"unit-health/internal/discovery"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

const canonicalTestingSchemaRel = "scenarios/test-genie/schemas/testing.schema.json"

// PreviewFix and ApplyFix implement Unit Health's deterministic low-risk fixes.
// They intentionally cover only config/projection edits that can be described as
// complete file before/after candidates. Dependency installation and behavioral
// test generation stay out of scope.
func (h *SharedHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req, false)
}

func (h *SharedHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req, true)
}

func (h *SharedHandler) fix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest], apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("fix request is required"))
	}
	scenario, root, err := h.resolveFixTarget(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	candidates, messages := collectFixCandidates(root, req.Msg.GetRuleIds())
	if apply {
		for _, c := range candidates {
			if err := applyCandidate(c); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			c.Applied = true
		}
	}
	if len(candidates) == 0 {
		messages = append(messages, "no deterministic Unit Health fixes available")
	}
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario:   scenario,
		Applied:    apply,
		Candidates: candidates,
		Messages:   messages,
	}), nil
}

func (h *SharedHandler) resolveFixTarget(ctx context.Context, req *scenariovalidationv1.FixRequest) (string, string, error) {
	if h == nil || h.handler == nil {
		return "", "", errors.New("unit validation handler not wired")
	}
	locator := discovery.Locator(discovery.DefaultLocator{})
	if h.handler.svc != nil && h.handler.svc.Locator != nil {
		locator = h.handler.svc.Locator
	}
	scenario, _, root, err := locator.Locate(ctx, req.GetScenario(), req.GetPath())
	if err != nil {
		return "", "", err
	}
	return scenario, root, nil
}

func collectFixCandidates(root string, ruleIDs []string) ([]*scenariovalidationv1.FixCandidate, []string) {
	allow := allowedRules(ruleIDs)
	var candidates []*scenariovalidationv1.FixCandidate
	var messages []string
	if allow(codeUnitPolicyInvalid) {
		candidates = append(candidates, testingSchemaCandidate(root)...)
	}
	if allow(codeUnitProjectionDrift) {
		candidates = append(candidates, packageJSONCandidates(root)...)
		candidates = append(candidates, viteConfigCandidates(root)...)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].FilePath != candidates[j].FilePath {
			return candidates[i].FilePath < candidates[j].FilePath
		}
		return candidates[i].RuleId < candidates[j].RuleId
	})
	if len(ruleIDs) > 0 && len(candidates) == 0 {
		messages = append(messages, "requested rule id(s) have no deterministic Unit Health fix")
	}
	return candidates, messages
}

func allowedRules(ruleIDs []string) func(string) bool {
	if len(ruleIDs) == 0 {
		return func(string) bool { return true }
	}
	allowed := map[string]bool{}
	for _, id := range ruleIDs {
		allowed[strings.TrimSpace(id)] = true
	}
	return func(rule string) bool { return allowed[rule] }
}

func testingSchemaCandidate(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, ".vrooli", "testing.json")
	before, ok := readFixFile(path)
	if !ok || !strings.Contains(before, "scripts/scenarios/testing/schemas/testing.schema.json") {
		return nil
	}
	repoRoot, err := findRepoRootFrom(root)
	if err != nil {
		return nil
	}
	target := filepath.Join(repoRoot, canonicalTestingSchemaRel)
	rel, err := filepath.Rel(filepath.Dir(path), target)
	if err != nil {
		return nil
	}
	after := strings.ReplaceAll(before, "../../../../scripts/scenarios/testing/schemas/testing.schema.json", filepath.ToSlash(rel))
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitPolicyInvalid, path, "Normalize testing.json schema reference to the active Test Genie schema.", before, after)}
}

func packageJSONCandidates(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, "ui", "package.json")
	before, ok := readFixFile(path)
	if !ok {
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(before), &doc); err != nil {
		return nil
	}
	scripts, _ := doc["scripts"].(map[string]any)
	if scripts == nil {
		scripts = map[string]any{}
		doc["scripts"] = scripts
	}
	changed := false
	if scriptValue(scripts["test"]) == "" || !strings.Contains(scriptValue(scripts["test"]), "vitest") {
		scripts["test"] = "vitest run"
		changed = true
	}
	if scriptValue(scripts["test:coverage"]) == "" || !strings.Contains(scriptValue(scripts["test:coverage"]), "coverage") {
		scripts["test:coverage"] = "vitest run --coverage"
		changed = true
	}
	if !changed {
		return nil
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil
	}
	after := string(raw) + "\n"
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Restore canonical Vitest test and coverage scripts.", before, after)}
}

func scriptValue(v any) string {
	s, _ := v.(string)
	return s
}

func viteConfigCandidates(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, "ui", "vite.config.ts")
	before, ok := readFixFile(path)
	if !ok {
		path = filepath.Join(root, "ui", "vite.config.js")
		before, ok = readFixFile(path)
		if !ok {
			return nil
		}
	}
	after := before
	after = ensureSetupFiles(after)
	after = raiseCoverageThresholds(after)
	if after == before {
		return nil
	}
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Restore canonical Vitest setupFiles and minimum 85% coverage thresholds where the test block already exists.", before, after)}
}

func ensureSetupFiles(src string) string {
	if strings.Contains(src, "setupFiles") || !strings.Contains(src, "test:") || !strings.Contains(src, "environment") || !strings.Contains(src, "jsdom") {
		return src
	}
	lines := strings.SplitAfter(src, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "environment") || !strings.Contains(line, "jsdom") {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines = append(lines[:i+1], append([]string{indent + "setupFiles: ['./src/test-setup.ts'],\n"}, lines[i+1:]...)...)
		return strings.Join(lines, "")
	}
	return src
}

func raiseCoverageThresholds(src string) string {
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		re := regexp.MustCompile(`(` + key + `\s*:\s*)([0-9]+(?:\.[0-9]+)?)`)
		src = re.ReplaceAllStringFunc(src, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) != 3 {
				return match
			}
			v, err := strconv.ParseFloat(parts[2], 64)
			if err != nil || v >= 85 {
				return match
			}
			return parts[1] + "85"
		})
	}
	return src
}

func applyCandidate(c *scenariovalidationv1.FixCandidate) error {
	current, err := os.ReadFile(c.GetFilePath())
	if err != nil {
		return fmt.Errorf("read %s before applying fix: %w", c.GetFilePath(), err)
	}
	if string(current) != c.GetBefore() {
		return fmt.Errorf("refusing to apply fix for %s: file changed since preview", c.GetFilePath())
	}
	if err := os.WriteFile(c.GetFilePath(), []byte(c.GetAfter()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", c.GetFilePath(), err)
	}
	return nil
}

func candidate(rule, path, desc, before, after string) *scenariovalidationv1.FixCandidate {
	return &scenariovalidationv1.FixCandidate{
		RuleId:      rule,
		FilePath:    path,
		Description: desc,
		Before:      before,
		After:       after,
	}
}

func readFixFile(path string) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func findRepoRootFrom(start string) (string, error) {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "scenarios", "test-genie", "schemas", "testing.schema.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root with %s not found from %s", canonicalTestingSchemaRel, start)
		}
		dir = parent
	}
}

const (
	codeUnitPolicyInvalid   = "UNIT_POLICY_PROFILE_INVALID"
	codeUnitProjectionDrift = "UNIT_POLICY_PROJECTION_DRIFT"
)
