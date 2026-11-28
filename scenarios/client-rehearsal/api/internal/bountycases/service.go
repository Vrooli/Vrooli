package bountycases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	maxEvaluatorFiles = 128
	maxEvaluatorBytes = 2 << 20
)

var (
	caseIDPattern            = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
	commitPattern            = regexp.MustCompile(`^[0-9a-f]{40}$`)
	repositorySegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
)

type service struct {
	repo        Repository
	trustedRoot string
}

// NewService binds registration to a single trusted corpus root. Registration
// is an owner operation and is deliberately not exposed as a public RPC.
func NewService(repo Repository, trustedRoot string) (Service, error) {
	if repo == nil {
		return nil, errors.New("bountycases.NewService requires repository")
	}
	trustedRoot = strings.TrimSpace(trustedRoot)
	if trustedRoot == "" || !filepath.IsAbs(trustedRoot) {
		return nil, errors.New("bountycases.NewService requires an absolute trusted corpus root")
	}
	root := filepath.Clean(trustedRoot)
	var err error
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve trusted corpus root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errors.New("trusted corpus root must be an existing directory")
	}
	return &service{repo: repo, trustedRoot: root}, nil
}

func (s *service) Register(ctx context.Context, bundle CaseBundle) (CaseBundle, error) {
	providedDigest := strings.ToLower(strings.TrimSpace(bundle.BundleDigest))
	if err := s.validate(&bundle); err != nil {
		return CaseBundle{}, err
	}
	canonical, err := canonicalBundleBytes(bundle)
	if err != nil {
		return CaseBundle{}, err
	}
	digest := sha256.Sum256(canonical)
	bundle.BundleDigest = hex.EncodeToString(digest[:])
	if providedDigest != "" && providedDigest != bundle.BundleDigest {
		return CaseBundle{}, ErrInvalid{Reason: "bundle_digest does not match canonical registered content"}
	}
	existing, err := s.repo.GetVersion(ctx, bundle.CaseID, bundle.BundleVersion)
	if err == nil {
		if existing.BundleDigest == bundle.BundleDigest {
			return existing, nil
		}
		return CaseBundle{}, ErrConflict{Reason: "an immutable bundle already exists at this case version"}
	}
	var missing ErrNotFound
	if !errors.As(err, &missing) {
		return CaseBundle{}, err
	}
	return s.repo.Register(ctx, bundle)
}

func (s *service) Resolve(ctx context.Context, lookup Lookup) (CaseBundle, error) {
	lookup.CaseID = strings.TrimSpace(lookup.CaseID)
	lookup.PublicURL = strings.TrimSpace(lookup.PublicURL)
	if (lookup.CaseID == "") == (lookup.PublicURL == "") {
		return CaseBundle{}, ErrInvalid{Reason: "exactly one of case_id or public_url is required"}
	}
	if lookup.CaseID != "" && !caseIDPattern.MatchString(lookup.CaseID) {
		return CaseBundle{}, ErrInvalid{Reason: "case_id has an invalid format"}
	}
	if lookup.PublicURL != "" && !validIssueURL(lookup.PublicURL) {
		return CaseBundle{}, ErrInvalid{Reason: "public_url must be an exact HTTPS GitHub issue URL"}
	}
	bundle, err := s.repo.Resolve(ctx, lookup)
	if err != nil {
		return CaseBundle{}, err
	}
	if err := s.verifyStoredBundle(bundle); err != nil {
		return CaseBundle{}, err
	}
	return bundle, nil
}

func (s *service) ResolveEvaluatorBundle(ctx context.Context, id string) (FrozenEvaluatorBundle, error) {
	caseID, version, digest, err := parseEvaluatorBundleID(strings.TrimSpace(id))
	if err != nil {
		return FrozenEvaluatorBundle{}, ErrInvalid{Reason: "evaluator_bundle_id has an invalid format"}
	}
	bundle, err := s.repo.GetVersion(ctx, caseID, version)
	if err != nil {
		return FrozenEvaluatorBundle{}, err
	}
	if !strings.EqualFold(bundle.BundleDigest, digest) {
		return FrozenEvaluatorBundle{}, ErrNotFound{Lookup: id}
	}
	if err := s.verifyStoredBundle(bundle); err != nil {
		return FrozenEvaluatorBundle{}, err
	}
	files, contentDigest, entrypointDigest, err := readEvaluatorFiles(bundle.FrozenEvaluatorRoot, bundle.FrozenEvaluatorEntrypoint)
	if err != nil {
		return FrozenEvaluatorBundle{}, ErrCorrupt{Reason: "frozen evaluator bundle cannot be read safely"}
	}
	if contentDigest != bundle.FrozenEvaluatorBundleDigest || entrypointDigest != bundle.FrozenEvaluatorDigest {
		return FrozenEvaluatorBundle{}, ErrCorrupt{Reason: "frozen evaluator bundle digest does not match registered content"}
	}
	return FrozenEvaluatorBundle{EvaluatorBundleID: evaluatorBundleID(bundle), CaseID: bundle.CaseID, BundleVersion: bundle.BundleVersion, Entrypoint: bundle.FrozenEvaluatorEntrypoint, EvaluatorDigest: bundle.FrozenEvaluatorDigest, ContentDigest: contentDigest, Files: files}, nil
}

func (s *service) ProjectForSolver(bundle CaseBundle) SolverProjection {
	return SolverProjection{
		CaseID: bundle.CaseID, Project: bundle.Project, Repository: bundle.Repository,
		Title: bundle.Title, Task: bundle.Task, PublicURL: bundle.PublicURL,
		IssueNumber: bundle.IssueNumber, SourceCommit: bundle.SourceCommit,
		SourceProjectRoot: bundle.SourceProjectRoot,
	}
}

func (s *service) validate(bundle *CaseBundle) error {
	if bundle.SchemaVersion != 1 || bundle.BundleVersion == 0 || bundle.BundleVersion > 1000 {
		return ErrInvalid{Reason: "schema_version must be 1 and bundle_version must be between 1 and 1000"}
	}
	bundle.CaseID = strings.TrimSpace(bundle.CaseID)
	bundle.Project = strings.TrimSpace(bundle.Project)
	bundle.Repository = strings.TrimSpace(bundle.Repository)
	bundle.Title = strings.TrimSpace(bundle.Title)
	bundle.Task = strings.TrimSpace(bundle.Task)
	bundle.PublicURL = strings.TrimSpace(bundle.PublicURL)
	bundle.PullRequestURL = strings.TrimSpace(bundle.PullRequestURL)
	bundle.SourceCommit = strings.ToLower(strings.TrimSpace(bundle.SourceCommit))
	bundle.ReferenceCommit = strings.ToLower(strings.TrimSpace(bundle.ReferenceCommit))
	bundle.FrozenEvaluatorDigest = strings.ToLower(strings.TrimSpace(bundle.FrozenEvaluatorDigest))
	bundle.FrozenEvaluatorBundleDigest = strings.ToLower(strings.TrimSpace(bundle.FrozenEvaluatorBundleDigest))
	bundle.FrozenEvaluatorEntrypoint = strings.TrimSpace(bundle.FrozenEvaluatorEntrypoint)
	bundle.TestGenieProjectID = strings.TrimSpace(bundle.TestGenieProjectID)
	bundle.BundleDigest = ""
	if !caseIDPattern.MatchString(bundle.CaseID) || bundle.Project == "" || bundle.Repository == "" || bundle.Title == "" || bundle.Task == "" {
		return ErrInvalid{Reason: "case_id, project, repository, title, and task are required"}
	}
	if len(bundle.Project) > 256 || len(bundle.Repository) > 256 || len(bundle.Title) > 512 || len(bundle.Task) > 32768 {
		return ErrInvalid{Reason: "case metadata exceeds its size limit"}
	}
	if !validRepository(bundle.Repository) || !validIssueURL(bundle.PublicURL) || !validPullRequestURL(bundle.PullRequestURL) {
		return ErrInvalid{Reason: "repository, public_url, or pull_request_url is invalid"}
	}
	if bundle.IssueNumber == 0 || bundle.PullRequestNumber == 0 ||
		!urlMatchesRepoAndNumber(bundle.PublicURL, bundle.Repository, "issues", bundle.IssueNumber) ||
		!urlMatchesRepoAndNumber(bundle.PullRequestURL, bundle.Repository, "pull", bundle.PullRequestNumber) {
		return ErrInvalid{Reason: "issue and pull request URLs must match their exact repository and numbers"}
	}
	if !commitPattern.MatchString(bundle.SourceCommit) || !commitPattern.MatchString(bundle.ReferenceCommit) {
		return ErrInvalid{Reason: "source_commit and reference_commit must be full lowercase git commit IDs"}
	}
	if bundle.TestGenieProjectID == "" || len(bundle.TestGenieProjectID) > 256 {
		return ErrInvalid{Reason: "test_genie_project_id is required and bounded"}
	}
	if err := validatePlan(bundle.TestGeniePlan); err != nil {
		return err
	}
	var err error
	if bundle.SourceProjectRoot, err = s.validateRoot(bundle.SourceProjectRoot); err != nil {
		return ErrInvalid{Reason: "source_project_root is not a safe trusted workspace: " + err.Error()}
	}
	if bundle.ReferenceProjectRoot, err = s.validateRoot(bundle.ReferenceProjectRoot); err != nil {
		return ErrInvalid{Reason: "reference_project_root is not a safe trusted workspace: " + err.Error()}
	}
	if bundle.FrozenEvaluatorRoot, err = s.validateRoot(bundle.FrozenEvaluatorRoot); err != nil {
		return ErrInvalid{Reason: "frozen_evaluator_root is not a safe trusted workspace: " + err.Error()}
	}
	contentDigest, entrypointDigest, err := evaluatorContentDigest(bundle.FrozenEvaluatorRoot, bundle.FrozenEvaluatorEntrypoint)
	if err != nil {
		return ErrInvalid{Reason: "frozen evaluator entrypoint is invalid: " + err.Error()}
	}
	if bundle.FrozenEvaluatorDigest != "" && !strings.EqualFold(bundle.FrozenEvaluatorDigest, entrypointDigest) {
		return ErrInvalid{Reason: "frozen_evaluator_digest does not match entrypoint bytes"}
	}
	if bundle.FrozenEvaluatorBundleDigest != "" && !strings.EqualFold(bundle.FrozenEvaluatorBundleDigest, contentDigest) {
		return ErrInvalid{Reason: "frozen_evaluator_bundle_digest does not match file manifest"}
	}
	bundle.FrozenEvaluatorDigest = entrypointDigest
	bundle.FrozenEvaluatorBundleDigest = contentDigest
	if bundle.BundleVersion == 0 || bundle.BundleVersion > 1000 {
		return ErrInvalid{Reason: "bundle_version must be between 1 and 1000"}
	}
	if bundle.SourceProjectRoot == bundle.ReferenceProjectRoot || bundle.SourceProjectRoot == bundle.FrozenEvaluatorRoot || bundle.ReferenceProjectRoot == bundle.FrozenEvaluatorRoot {
		return ErrInvalid{Reason: "source, reference, and evaluator roots must be distinct"}
	}
	return nil
}

func (s *service) validateRoot(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("path must be absolute")
	}
	clean := filepath.Clean(path)
	real, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.trustedRoot, real)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the configured corpus root")
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", errors.New("path must resolve to an existing directory")
	}
	return real, nil
}

func (s *service) verifyStoredBundle(bundle CaseBundle) error {
	if bundle.SchemaVersion != 1 || bundle.BundleVersion == 0 || !validSHA256(bundle.BundleDigest) {
		return ErrCorrupt{Reason: "registered bundle version or digest is invalid"}
	}
	got := bundle.BundleDigest
	bundle.BundleDigest = ""
	data, err := canonicalBundleBytes(bundle)
	if err != nil {
		return ErrCorrupt{Reason: "registered bundle cannot be encoded"}
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(got, hex.EncodeToString(sum[:])) {
		return ErrCorrupt{Reason: "registered bundle digest does not match stored content"}
	}
	for _, path := range []string{bundle.SourceProjectRoot, bundle.ReferenceProjectRoot, bundle.FrozenEvaluatorRoot} {
		if _, err := s.validateRoot(path); err != nil {
			return ErrCorrupt{Reason: "registered workspace root is no longer safe"}
		}
	}
	contentDigest, entrypointDigest, err := evaluatorContentDigest(bundle.FrozenEvaluatorRoot, bundle.FrozenEvaluatorEntrypoint)
	if err != nil || contentDigest != bundle.FrozenEvaluatorBundleDigest || entrypointDigest != bundle.FrozenEvaluatorDigest {
		return ErrCorrupt{Reason: "frozen evaluator content no longer matches its digest"}
	}
	return nil
}

func canonicalBundleBytes(bundle CaseBundle) ([]byte, error) {
	bundle.BundleDigest = ""
	return json.Marshal(bundle)
}

func validatePlan(plan TestGeniePlan) error {
	if plan.ValidationPurpose != "VALIDATION_PURPOSE_REGRESSION_CURRENT" || plan.RequiredStrength != "VALIDATION_STRENGTH_TARGETED" {
		return ErrInvalid{Reason: "F9 plan requires closed Regression Current purpose and Targeted strength values"}
	}
	if len(plan.Phases) != 0 {
		return ErrInvalid{Reason: "F9 frozen replay uses its sealed evaluator and cannot select generic Test Genie phases"}
	}
	if plan.DeadlinePolicy.QueueBudgetMilliseconds == 0 || plan.DeadlinePolicy.QueueBudgetMilliseconds > 3_600_000 || plan.DeadlinePolicy.ExecutionBudgetMilliseconds == 0 || plan.DeadlinePolicy.ExecutionBudgetMilliseconds > 3_600_000 || plan.DeadlinePolicy.MaximumAttempts != 1 {
		return ErrInvalid{Reason: "frozen deadline policy requires bounded positive budgets and one attempt"}
	}
	if !plan.EvidencePolicy.RequireSourceSnapshot || plan.EvidencePolicy.AllowAuthorizedDegradation {
		return ErrInvalid{Reason: "frozen evidence policy requires source snapshot and forbids authorized degradation"}
	}
	if len(plan.EvidencePolicy.RequiredEvidenceKinds) == 0 || len(plan.EvidencePolicy.RequiredEvidenceKinds) > 8 {
		return ErrInvalid{Reason: "frozen evidence policy requires 1..8 evidence kinds"}
	}
	seenKinds := make(map[string]struct{}, len(plan.EvidencePolicy.RequiredEvidenceKinds))
	for i, kind := range plan.EvidencePolicy.RequiredEvidenceKinds {
		kind = strings.TrimSpace(kind)
		if kind == "" || len(kind) > 128 {
			return ErrInvalid{Reason: "frozen evidence kinds must be nonempty and bounded"}
		}
		if _, ok := seenKinds[kind]; ok {
			return ErrInvalid{Reason: "frozen evidence kinds must be unique"}
		}
		seenKinds[kind] = struct{}{}
		plan.EvidencePolicy.RequiredEvidenceKinds[i] = kind
	}
	return nil
}

func validRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && repositorySegmentPattern.MatchString(parts[0]) && repositorySegmentPattern.MatchString(parts[1])
}

func validIssueURL(raw string) bool       { return validGitHubURL(raw, "issues") }
func validPullRequestURL(raw string) bool { return validGitHubURL(raw, "pull") }

func validGitHubURL(raw, kind string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != kind {
		return false
	}
	if u.EscapedPath() != u.Path || u.Path != "/"+strings.Join(parts, "/") {
		return false
	}
	n, err := parsePositiveDecimal(parts[3])
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == parts[3]
}

func urlMatchesRepoAndNumber(raw, repository, kind string, number uint64) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0]+"/"+parts[1] != repository || parts[2] != kind {
		return false
	}
	n, err := parsePositiveDecimal(parts[3])
	return err == nil && n == number
}

func parsePositiveDecimal(s string) (uint64, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 {
		return 0, errors.New("not a positive decimal")
	}
	return n, nil
}

func validSHA256(v string) bool {
	if len(v) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}

// evaluatorContentDigest hashes a stable manifest of every regular evaluator
// file. The entrypoint digest remains the SHA-256 of that file's exact bytes.
func evaluatorContentDigest(root, entrypoint string) (string, string, error) {
	if entrypoint == "" || filepath.IsAbs(entrypoint) || strings.Contains(entrypoint, "\\") || path.Clean(entrypoint) != entrypoint || entrypoint == "." || entrypoint == ".." || strings.HasPrefix(entrypoint, "../") {
		return "", "", errors.New("entrypoint must be a clean relative path")
	}
	joined := filepath.Join(root, filepath.FromSlash(entrypoint))
	if !within(root, joined) {
		return "", "", errors.New("entrypoint escapes evaluator root")
	}
	_, contentDigest, entryDigest, err := walkEvaluatorFiles(root, entrypoint, false)
	return contentDigest, entryDigest, err
}

type evaluatorManifestEntry struct {
	RelativePath string `json:"relativePath"`
	SHA256       string `json:"sha256"`
	Size         int64  `json:"size"`
}

func readEvaluatorFiles(root, entrypoint string) ([]EvaluatorFile, string, string, error) {
	files, contentDigest, entryDigest, err := walkEvaluatorFiles(root, entrypoint, true)
	return files, contentDigest, entryDigest, err
}

func walkEvaluatorFiles(root, entrypoint string, includeBytes bool) ([]EvaluatorFile, string, string, error) {
	files := make([]EvaluatorFile, 0)
	manifest := make([]evaluatorManifestEntry, 0)
	var total int64
	entryDigest := ""
	err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("evaluator bundles cannot contain symlinks")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("evaluator bundles may contain only regular files")
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || path.Clean(rel) != rel {
			return errors.New("evaluator file path escapes root")
		}
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		if int64(len(contents)) > maxEvaluatorBytes || total+int64(len(contents)) > maxEvaluatorBytes {
			return errors.New("evaluator bundle exceeds byte limit")
		}
		sum := sha256.Sum256(contents)
		digest := hex.EncodeToString(sum[:])
		manifest = append(manifest, evaluatorManifestEntry{RelativePath: rel, SHA256: digest, Size: int64(len(contents))})
		if rel == entrypoint {
			entryDigest = digest
		}
		if includeBytes {
			files = append(files, EvaluatorFile{RelativePath: rel, Content: contents, SHA256: digest})
		}
		total += int64(len(contents))
		if len(manifest) > maxEvaluatorFiles {
			return errors.New("evaluator bundle exceeds file-count limit")
		}
		return nil
	})
	if err != nil {
		return nil, "", "", err
	}
	if len(manifest) == 0 || entryDigest == "" {
		return nil, "", "", errors.New("evaluator entrypoint is absent")
	}
	sort.Slice(manifest, func(i, j int) bool { return manifest[i].RelativePath < manifest[j].RelativePath })
	sort.Slice(files, func(i, j int) bool { return files[i].RelativePath < files[j].RelativePath })
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, "", "", err
	}
	sum := sha256.Sum256(manifestJSON)
	return files, hex.EncodeToString(sum[:]), entryDigest, nil
}

func evaluatorBundleID(bundle CaseBundle) string {
	return fmt.Sprintf("%s@%d:%s", bundle.CaseID, bundle.BundleVersion, bundle.BundleDigest)
}

func parseEvaluatorBundleID(id string) (string, uint32, string, error) {
	caseID, versionAndDigest, ok := strings.Cut(id, "@")
	if !ok || !caseIDPattern.MatchString(caseID) {
		return "", 0, "", errors.New("invalid case id")
	}
	versionText, digest, ok := strings.Cut(versionAndDigest, ":")
	if !ok || !validSHA256(digest) {
		return "", 0, "", errors.New("invalid digest")
	}
	version, err := strconv.ParseUint(versionText, 10, 32)
	if err != nil || version == 0 || strconv.FormatUint(version, 10) != versionText {
		return "", 0, "", errors.New("invalid bundle version")
	}
	return caseID, uint32(version), strings.ToLower(digest), nil
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type ErrInvalid struct{ Reason string }

func (e ErrInvalid) Error() string { return e.Reason }

type ErrNotFound struct{ Lookup string }

func (e ErrNotFound) Error() string { return "registered bounty case not found: " + e.Lookup }

type ErrConflict struct{ Reason string }

func (e ErrConflict) Error() string { return e.Reason }

type ErrCorrupt struct{ Reason string }

func (e ErrCorrupt) Error() string { return "registered bounty case is corrupt: " + e.Reason }
