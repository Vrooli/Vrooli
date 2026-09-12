package impact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"proto-health/internal/protosurface"

	"github.com/vrooli/api-core/discovery"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	impactv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact"
)

type Service struct {
	RepoRoot  string
	Loader    SurfaceLoader
	Runner    BreakingRunner
	Baselines BaselineLister
}

type SurfaceLoader interface {
	LoadScenario(scenario string) (protosurface.Surface, error)
}

type BreakingRunner interface {
	RunBreaking(ctx context.Context, repoRoot, scenario, baselineInput string) ([]byte, error)
}

type BaselineLister interface {
	ListBaselines(ctx context.Context, scenario, branch string) ([]*baselinesv1.BaselineManifest, error)
}

type commandRunner struct{}

type resolvedScope struct {
	input          string
	kind           string
	sha            string
	baselineName   string
	commitsSince   int32
	likelyStale    bool
	fallbackReason string
}

func New(repoRoot string, loader SurfaceLoader) *Service {
	return &Service{
		RepoRoot: repoRoot,
		Loader:   loader,
		Runner:   commandRunner{},
	}
}

func (s *Service) GetImpact(ctx context.Context, scenario, against string) (*impactv1.ImpactReport, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	if s.RepoRoot == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	if s.Runner == nil {
		s.Runner = commandRunner{}
	}

	scope, err := s.resolveScope(ctx, scenario, against)
	if err != nil {
		return nil, err
	}
	baselineInput, cleanup, err := extractBaselineProtoPackage(ctx, s.RepoRoot, scope.sha)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	output, err := s.Runner.RunBreaking(ctx, s.RepoRoot, scenario, baselineInput)
	if err != nil && len(output) == 0 {
		return nil, err
	}

	var surface protosurface.Surface
	if s.Loader != nil {
		if surface, loadErr := s.Loader.LoadScenario(scenario); loadErr == nil {
			return reportFromBreakingOutput(scenario, scope, output, surface), nil
		}
	}
	return reportFromBreakingOutput(scenario, scope, output, surface), nil
}

func (s *Service) resolveScope(ctx context.Context, scenario, input string) (resolvedScope, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		scope, err := s.resolveDefaultScope(ctx, scenario)
		if err == nil {
			return scope, nil
		}
		mergeBase, mergeErr := gitMergeBase(ctx, s.RepoRoot)
		if mergeErr != nil {
			return resolvedScope{}, err
		}
		scope = resolvedScope{input: "merge-base", kind: "merge-base", sha: mergeBase, fallbackReason: err.Error()}
		return scope, nil
	}
	switch input {
	case "HEAD":
		sha, err := gitRevParse(ctx, s.RepoRoot, "HEAD")
		return resolvedScope{input: input, kind: "head", sha: sha}, err
	case "master":
		sha, err := gitRevParse(ctx, s.RepoRoot, "master")
		return resolvedScope{input: input, kind: "master", sha: sha}, err
	case "merge-base":
		sha, err := gitMergeBase(ctx, s.RepoRoot)
		return resolvedScope{input: input, kind: "merge-base", sha: sha}, err
	default:
		if name, ok := strings.CutPrefix(input, "baseline:"); ok {
			return s.resolveNamedBaseline(ctx, scenario, strings.TrimSpace(name))
		}
		sha, err := gitRevParse(ctx, s.RepoRoot, input)
		return resolvedScope{input: input, kind: "git-ref", sha: sha}, err
	}
}

func (s *Service) resolveDefaultScope(ctx context.Context, scenario string) (resolvedScope, error) {
	if s.Baselines == nil {
		return resolvedScope{}, fmt.Errorf("git-control-tower baseline resolver is not wired")
	}
	branch, err := gitCurrentBranch(ctx, s.RepoRoot)
	if err != nil {
		return resolvedScope{}, err
	}
	baselines, err := s.Baselines.ListBaselines(ctx, scenario, branch)
	if err != nil {
		if discovery.IsScenarioNotRunning(err) {
			return resolvedScope{}, fmt.Errorf("git-control-tower is not running")
		}
		return resolvedScope{}, fmt.Errorf("list git-control-tower baselines: %w", err)
	}
	baseline := newestBaselineWithSHA(baselines)
	if baseline == nil {
		return resolvedScope{}, fmt.Errorf("no git-control-tower baseline found for %s on %s", scenario, branch)
	}
	return s.scopeFromBaseline(ctx, baseline)
}

func (s *Service) resolveNamedBaseline(ctx context.Context, scenario, name string) (resolvedScope, error) {
	if name == "" {
		return resolvedScope{}, fmt.Errorf("baseline name is required")
	}
	if s.Baselines == nil {
		return resolvedScope{}, fmt.Errorf("git-control-tower baseline resolver is not wired")
	}
	branch, err := gitCurrentBranch(ctx, s.RepoRoot)
	if err != nil {
		return resolvedScope{}, err
	}
	baselines, err := s.Baselines.ListBaselines(ctx, scenario, branch)
	if err != nil {
		return resolvedScope{}, fmt.Errorf("list git-control-tower baselines: %w", err)
	}
	for _, baseline := range baselines {
		if baseline.GetName() == name {
			return s.scopeFromBaseline(ctx, baseline)
		}
	}
	return resolvedScope{}, fmt.Errorf("baseline %q not found for %s on %s", name, scenario, branch)
}

func (s *Service) scopeFromBaseline(ctx context.Context, baseline *baselinesv1.BaselineManifest) (resolvedScope, error) {
	sha := baseline.GetGit().GetSha()
	if strings.TrimSpace(sha) == "" {
		return resolvedScope{}, fmt.Errorf("baseline %q has no git sha", baseline.GetName())
	}
	commits, err := gitCommitsSince(ctx, s.RepoRoot, sha)
	if err != nil {
		return resolvedScope{}, err
	}
	return resolvedScope{
		input:        "baseline:" + baseline.GetName(),
		kind:         "baseline",
		sha:          sha,
		baselineName: baseline.GetName(),
		commitsSince: commits,
		likelyStale:  commits > 0,
	}, nil
}

func newestBaselineWithSHA(baselines []*baselinesv1.BaselineManifest) *baselinesv1.BaselineManifest {
	var best *baselinesv1.BaselineManifest
	for _, baseline := range baselines {
		if baseline.GetGit().GetSha() == "" {
			continue
		}
		if best == nil || baseline.GetCreatedAt() > best.GetCreatedAt() {
			best = baseline
		}
	}
	return best
}

func reportFromBreakingOutput(scenario string, scope resolvedScope, output []byte, surface protosurface.Surface) *impactv1.ImpactReport {
	stabilityByFile := map[string]string{}
	for _, file := range surface.Files {
		stabilityByFile[file.Path] = file.Stability
	}
	changes := parseBreakingOutput(output, stabilityByFile)
	report := &impactv1.ImpactReport{
		Scenario:             scenario,
		Scope:                scope.input,
		ScopeKind:            scope.kind,
		BaselineSha:          scope.sha,
		BaselineName:         scope.baselineName,
		CommitsSinceBaseline: scope.commitsSince,
		LikelyStale:          scope.likelyStale,
		FallbackReason:       scope.fallbackReason,
		Changes:              changes,
	}
	for _, change := range changes {
		if change.GetWireBreaking() {
			report.WireBreakingCount++
		}
		if change.GetJsonBreaking() {
			report.JsonBreakingCount++
		}
	}
	attachConsumerReconciliation(report, surface)
	return report
}

func (commandRunner) RunBreaking(ctx context.Context, repoRoot, scenario, baselineInput string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "buf", "breaking", "packages/proto",
		"--against", baselineInput,
		"--error-format=json",
	)
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := bytes.TrimSpace(stdout.Bytes())
	if err != nil && len(out) == 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("buf breaking: %s", msg)
	}
	return out, err
}

func gitRevParse(ctx context.Context, repoRoot, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", ref+"^{commit}")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve git ref %q: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitCurrentBranch(ctx context.Context, repoRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve current git branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", fmt.Errorf("current git branch is detached")
	}
	return branch, nil
}

func gitMergeBase(ctx context.Context, repoRoot string) (string, error) {
	for _, ref := range []string{"master", "origin/master"} {
		cmd := exec.CommandContext(ctx, "git", "merge-base", "HEAD", ref)
		cmd.Dir = repoRoot
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", fmt.Errorf("resolve merge-base against master or origin/master")
}

func gitCommitsSince(ctx context.Context, repoRoot, sha string) (int32, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", sha+"..HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("count commits since %s: %w", sha, err)
	}
	var count int32
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count); err != nil {
		return 0, fmt.Errorf("parse commits since %s: %w", sha, err)
	}
	return count, nil
}

func extractBaselineProtoPackage(ctx context.Context, repoRoot, sha string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "proto-health-impact-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "packages", "proto")
	if err := os.MkdirAll(path, 0o755); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("create baseline package dir: %w", err)
	}

	if err := copyCurrentBufInputs(repoRoot, path); err != nil {
		cleanup()
		return "", func() {}, err
	}
	cmd := exec.CommandContext(ctx, "git", "archive", sha+":packages/proto", "schemas")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("extract baseline schemas for %s: %w", sha, err)
	}
	tar := exec.CommandContext(ctx, "tar", "-x", "-C", path)
	tar.Stdin = bytes.NewReader(out)
	if err := tar.Run(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("unpack baseline schemas: %w", err)
	}
	return path, cleanup, nil
}

func copyCurrentBufInputs(repoRoot, destProtoPath string) error {
	currentProtoPath := filepath.Join(repoRoot, "packages", "proto")
	for _, name := range []string{"buf.yaml", "buf.lock"} {
		if err := copyFile(filepath.Join(currentProtoPath, name), filepath.Join(destProtoPath, name)); err != nil {
			return fmt.Errorf("copy current %s: %w", name, err)
		}
	}
	if err := copyDir(filepath.Join(currentProtoPath, "vendor"), filepath.Join(destProtoPath, "vendor")); err != nil {
		return fmt.Errorf("copy current vendored proto dependencies: %w", err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func attachConsumerReconciliation(report *impactv1.ImpactReport, surface protosurface.Surface) {
	consumersByFile := consumersForFiles(surface.CrossScenarioImports)
	seenReportConsumers := map[string]bool{}
	for _, change := range report.GetChanges() {
		if !changeRequiresConsumerReconciliation(change) {
			continue
		}
		for _, consumer := range consumersByFile[change.GetFile()] {
			change.UnreconciledConsumers = append(change.UnreconciledConsumers, consumer)
			key := consumer.GetScenario() + "\x00" + consumer.GetFromFile() + "\x00" + consumer.GetToFile()
			if !seenReportConsumers[key] {
				report.UnreconciledConsumers = append(report.UnreconciledConsumers, consumer)
				seenReportConsumers[key] = true
			}
		}
		if strings.EqualFold(change.GetStability(), "stable") && len(change.GetUnreconciledConsumers()) > 0 {
			report.StableUnreconciledBreakingCount++
		}
	}
	sort.Slice(report.UnreconciledConsumers, func(i, j int) bool {
		left, right := report.UnreconciledConsumers[i], report.UnreconciledConsumers[j]
		if left.GetScenario() != right.GetScenario() {
			return left.GetScenario() < right.GetScenario()
		}
		if left.GetFromFile() != right.GetFromFile() {
			return left.GetFromFile() < right.GetFromFile()
		}
		return left.GetToFile() < right.GetToFile()
	})
	report.UnreconciledConsumerCount = int32(len(report.UnreconciledConsumers))
}

func consumersForFiles(imports []protosurface.Import) map[string][]*impactv1.ImpactConsumer {
	out := map[string][]*impactv1.ImpactConsumer{}
	seen := map[string]bool{}
	for _, imp := range imports {
		if imp.ToFile == "" || imp.FromScenario == "" {
			continue
		}
		key := imp.ToFile + "\x00" + imp.FromScenario + "\x00" + imp.FromFile
		if seen[key] {
			continue
		}
		seen[key] = true
		out[imp.ToFile] = append(out[imp.ToFile], &impactv1.ImpactConsumer{
			Scenario:     imp.FromScenario,
			FromFile:     imp.FromFile,
			ToFile:       imp.ToFile,
			Unreconciled: true,
		})
	}
	for file := range out {
		sort.Slice(out[file], func(i, j int) bool {
			if out[file][i].GetScenario() != out[file][j].GetScenario() {
				return out[file][i].GetScenario() < out[file][j].GetScenario()
			}
			return out[file][i].GetFromFile() < out[file][j].GetFromFile()
		})
	}
	return out
}

func changeRequiresConsumerReconciliation(change *impactv1.ImpactChange) bool {
	if !change.GetWireBreaking() && !change.GetJsonBreaking() {
		return false
	}
	return !strings.EqualFold(change.GetStability(), "experimental")
}

type bufFinding struct {
	Path      string `json:"path"`
	Message   string `json:"message"`
	StartLine int32  `json:"start_line"`
}

func parseBreakingOutput(output []byte, stabilityByFile map[string]string) []*impactv1.ImpactChange {
	findings := decodeBufFindings(output)
	changes := make([]*impactv1.ImpactChange, 0, len(findings))
	for _, finding := range findings {
		file := normalizeBufPath(finding.Path)
		if len(stabilityByFile) > 0 && !strings.HasPrefix(file, stabilityScenarioPrefix(stabilityByFile)) {
			continue
		}
		kind := classifyKind(finding.Message)
		change := &impactv1.ImpactChange{
			File:         file,
			Path:         findingPath(finding),
			Kind:         kind,
			WireBreaking: true,
			JsonBreaking: jsonBreaking(kind),
			Stability:    stabilityByFile[file],
			Message:      finding.Message,
		}
		changes = append(changes, change)
	}
	return changes
}

func stabilityScenarioPrefix(stabilityByFile map[string]string) string {
	for file := range stabilityByFile {
		if scenario, _, ok := strings.Cut(file, "/"); ok {
			return scenario + "/"
		}
	}
	return ""
}

func decodeBufFindings(output []byte) []bufFinding {
	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return nil
	}
	var direct []bufFinding
	if err := json.Unmarshal(output, &direct); err == nil {
		return direct
	}
	var wrapped struct {
		FileAnnotations []bufFinding `json:"file_annotations"`
	}
	if err := json.Unmarshal(output, &wrapped); err == nil && len(wrapped.FileAnnotations) > 0 {
		return wrapped.FileAnnotations
	}
	lines := strings.Split(string(output), "\n")
	findings := make([]bufFinding, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		findings = append(findings, parseTextFinding(line))
	}
	return findings
}

var textFindingRE = regexp.MustCompile(`^([^:]+\.proto)(?::\d+:\d+)?:\s*(.+)$`)

func parseTextFinding(line string) bufFinding {
	matches := textFindingRE.FindStringSubmatch(line)
	if len(matches) == 3 {
		return bufFinding{Path: matches[1], Message: matches[2]}
	}
	return bufFinding{Message: line}
}

func normalizeBufPath(path string) string {
	path = strings.TrimSpace(filepath.ToSlash(path))
	path = strings.TrimPrefix(path, "schemas/")
	return path
}

func findingPath(finding bufFinding) string {
	if finding.StartLine > 0 {
		return fmt.Sprintf("%s:%d", normalizeBufPath(finding.Path), finding.StartLine)
	}
	return normalizeBufPath(finding.Path)
}

func classifyKind(message string) impactv1.ImpactChangeKind {
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "field") && strings.Contains(msg, "number"):
		return impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RENUMBER
	case strings.Contains(msg, "type") || strings.Contains(msg, "kind"):
		return impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RETYPE
	case strings.Contains(msg, "rename") || strings.Contains(msg, "json name"):
		return impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RENAME
	case strings.Contains(msg, "delete") || strings.Contains(msg, "remove") || strings.Contains(msg, "previously present"):
		return impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_REMOVE
	default:
		return impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_UNSPECIFIED
	}
}

func jsonBreaking(kind impactv1.ImpactChangeKind) bool {
	switch kind {
	case impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RENUMBER:
		return false
	default:
		return true
	}
}
