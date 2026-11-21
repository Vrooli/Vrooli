package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
)

// Gateway role literals are recognized only in likely Gateway caller source.
var (
	goGatewayRoleLiteral  = regexp.MustCompile(`\bRole\s*:\s*"([a-z][a-z0-9_-]*\.[a-z0-9_.-]+)"`)
	goDefaultRoleLiteral  = regexp.MustCompile(`(?i)\bdefault\w*role\s*:?=\s*"([a-z][a-z0-9_-]*\.[a-z0-9_.-]+)"`)
	goRoleConstant        = regexp.MustCompile(`\b[A-Z][A-Za-z0-9]*Role\s*=\s*"([a-z][a-z0-9_-]*\.[a-z0-9_.-]+)"`)
	implicitLLMRerankRole = regexp.MustCompile(`\bNewLLMRerankerWithRunner\s*\(\s*""\s*,`)
	jsonGatewayRole       = regexp.MustCompile(`(?i)["']role["']\s*:\s*["']([a-z][a-z0-9_-]*\.[a-z0-9_.-]+)["']`)
	jsGatewayRole         = regexp.MustCompile(`(?i)\brole\s*:\s*["']([a-z][a-z0-9_-]*\.[a-z0-9_.-]+)["']`)
)

// scanGatewayRoleReachability verifies literal role IDs in Gateway callers
// against the authoritative task-role catalog. It only inspects this scenario,
// excludes the Gateway implementation itself, and ignores indirect/dynamic role
// values that cannot be safely resolved by a source lint.
func scanGatewayRoleReachability(ctx context.Context, scenarioRoot string) ([]*conformancev1.ConformanceFinding, error) {
	known, shouldScan, err := loadGatewayRoleCatalog(scenarioRoot)
	if err != nil || !shouldScan {
		return nil, err
	}
	var findings []*conformancev1.ConformanceFinding
	err = filepath.WalkDir(scenarioRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		fileFindings, err := scanGatewayRoleFile(path, scenarioRoot, known)
		if err != nil {
			return err
		}
		findings = append(findings, fileFindings...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].GetPath() < findings[j].GetPath() })
	return findings, nil
}

func loadGatewayRoleCatalog(scenarioRoot string) (map[string]struct{}, bool, error) {
	catalogPath, scenarioID := locateGatewayRoleCatalog(scenarioRoot)
	if catalogPath == "" || scenarioID == "ai-gateway" {
		return nil, false, nil
	}
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, false, fmt.Errorf("read AI Gateway task-role catalog: %w", err)
	}
	var catalog struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, false, fmt.Errorf("decode AI Gateway task-role catalog: %w", err)
	}
	if len(catalog.Roles) == 0 {
		return nil, false, fmt.Errorf("AI Gateway task-role catalog declares no roles")
	}
	known := make(map[string]struct{}, len(catalog.Roles))
	for role := range catalog.Roles {
		known[role] = struct{}{}
	}
	return known, true, nil
}

func scanGatewayRoleFile(path, scenarioRoot string, known map[string]struct{}) ([]*conformancev1.ConformanceFinding, error) {
	if !scanFile(path) {
		return nil, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	source := string(body)
	if !isGatewayInferenceCaller(source) {
		return nil, nil
	}
	rel, err := filepath.Rel(scenarioRoot, path)
	if err != nil {
		return nil, err
	}
	roleFindings := gatewayRoleLiteralFindings(filepath.ToSlash(rel), gatewayRoleLines(source), known)
	return append(roleFindings, implicitGatewayRoleFindings(filepath.ToSlash(rel), source)...), nil
}

func gatewayRoleLines(source string) map[int]map[string]struct{} {
	roleLines := map[int]map[string]struct{}{}
	patterns := []*regexp.Regexp{goGatewayRoleLiteral, goDefaultRoleLiteral, goRoleConstant, jsonGatewayRole, jsGatewayRole}
	for lineNo, line := range strings.Split(source, "\n") {
		for _, pattern := range patterns {
			for _, match := range pattern.FindAllStringSubmatch(line, -1) {
				if roleLines[lineNo+1] == nil {
					roleLines[lineNo+1] = map[string]struct{}{}
				}
				roleLines[lineNo+1][match[1]] = struct{}{}
			}
		}
	}
	return roleLines
}

func gatewayRoleLiteralFindings(rel string, roleLines map[int]map[string]struct{}, known map[string]struct{}) []*conformancev1.ConformanceFinding {
	var findings []*conformancev1.ConformanceFinding
	for lineNo, roles := range roleLines {
		for role := range roles {
			if _, ok := known[role]; ok {
				continue
			}
			findings = append(findings, conformanceFinding(
				"ai.gateway_role_unreachable", "high", fmt.Sprintf("%s:%d", rel, lineNo),
				"A literal AI Gateway role is absent from the authoritative task-role catalog.",
				"Select a declared Gateway role or update the owner-reviewed role catalog through its policy owner.",
			))
		}
	}
	return findings
}

func implicitGatewayRoleFindings(rel, source string) []*conformancev1.ConformanceFinding {
	var findings []*conformancev1.ConformanceFinding
	for lineNo, line := range strings.Split(source, "\n") {
		if implicitLLMRerankRole.MatchString(line) {
			findings = append(findings, conformanceFinding(
				"ai.gateway_role_implicit", "high", fmt.Sprintf("%s:%d", rel, lineNo+1),
				"The typed Gateway LLM reranker inherits an implicit shared-library role instead of naming a catalog task role.",
				"Pass an owner-selected role declared in the AI Gateway task-role catalog; do not inherit a resource compatibility default.",
			))
		}
	}
	return findings
}

func isGatewayInferenceCaller(source string) bool {
	lower := strings.ToLower(source)
	return strings.Contains(lower, "ai-gateway/v1/inference") ||
		strings.Contains(lower, "ai_gateway.v1.inference") ||
		strings.Contains(lower, "ai-gateway/inference/run") ||
		strings.Contains(lower, "ai-gateway/inference/decide") ||
		strings.Contains(lower, "ai-gateway/inference/embed") ||
		(strings.Contains(lower, "routingconnect.") && strings.Contains(lower, "executeroute")) ||
		strings.Contains(lower, "aigateway.newtextgenerator") ||
		strings.Contains(lower, "inferenceconnect.")
}

func locateGatewayRoleCatalog(scenarioRoot string) (catalogPath, scenarioID string) {
	for dir := scenarioRoot; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "scenarios", "ai-gateway", "config", "inference-role-catalog.json")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			id := filepath.Base(scenarioRoot)
			return candidate, id
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
	}
}
