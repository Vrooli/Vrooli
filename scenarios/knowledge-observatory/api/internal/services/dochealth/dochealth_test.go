package dochealth

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// --- test helpers ----------------------------------------------------------

// fakeDoer returns canned responses by URL prefix, letting tests bypass real
// network calls for the external-link probe.
type fakeDoer struct {
	resp   map[string]*http.Response
	err    map[string]error
	called []string
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.called = append(f.called, req.URL.String())
	if e, ok := f.err[req.URL.String()]; ok {
		return nil, e
	}
	if r, ok := f.resp[req.URL.String()]; ok {
		return r, nil
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
}

type fakeCommandValidator struct {
	results map[string]CommandReferenceResult
	calls   []CommandReferenceRequest
}

func (f *fakeCommandValidator) ValidateCommandReference(ctx context.Context, req CommandReferenceRequest) (CommandReferenceResult, error) {
	_ = ctx
	f.calls = append(f.calls, req)
	if res, ok := f.results[req.CommandText]; ok {
		return res, nil
	}
	return CommandReferenceResult{CommandText: req.CommandText, Verdict: "unknown", ValidationLevel: "parsed"}, nil
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newCfg() effective {
	return defaultStaticConfig().withOptions(DocHealthOptions{})
}

func docsManifestRel(t *testing.T) string {
	t.Helper()
	rel, err := repocontract.ScenarioDocsManifestRel("")
	if err != nil {
		t.Fatalf("resolve docs manifest rel: %v", err)
	}
	return rel
}

// --- markdown / mermaid ----------------------------------------------------

func TestInspectMarkdownFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	writeTestFile(t, path, "# heading\n\nsome text [home](other.md)\n")

	findings, sum, links, ioErrs := inspectMarkdownFile(path, newCfg())
	if len(ioErrs) != 0 {
		t.Fatalf("unexpected io errors: %v", ioErrs)
	}
	if sum.MarkdownFailures != 0 {
		t.Fatalf("expected 0 failures, got %d", sum.MarkdownFailures)
	}
	if len(links) != 1 || links[0].Dest != "other.md" {
		t.Fatalf("expected 1 link to other.md, got %#v", links)
	}
	for _, f := range findings {
		if f.Severity == SeverityFailure {
			t.Fatalf("unexpected failure finding: %+v", f)
		}
	}
}

func TestInspectMarkdownFile_UnclosedFenceFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.md")
	writeTestFile(t, path, "# x\n```go\nfmt.Println\n")

	findings, sum, _, _ := inspectMarkdownFile(path, newCfg())
	if sum.MarkdownFailures != 1 {
		t.Fatalf("expected 1 markdown failure, got %d", sum.MarkdownFailures)
	}
	if !hasFinding(findings, "markdown_unclosed_fence") {
		t.Fatalf("expected markdown_unclosed_fence, got %+v", findings)
	}
}

func TestValidateMermaid_StrictFailsInvalid(t *testing.T) {
	cfg := newCfg()
	cfg.mermaidStrict = true
	var findings []Finding
	var sum fileMetrics
	validateMermaidBlock("x.md", 5, "not a valid diagram (((", true, &findings, &sum)
	if sum.MermaidFailures != 1 {
		t.Fatalf("expected mermaid failure, got %+v", sum)
	}
	if findings[0].Severity != SeverityFailure {
		t.Fatalf("expected failure severity")
	}
	_ = cfg
}

func TestValidateMermaid_NonStrictWarnsInvalid(t *testing.T) {
	var findings []Finding
	var sum fileMetrics
	validateMermaidBlock("x.md", 1, "garbage", false, &findings, &sum)
	if sum.MarkdownWarnings != 1 {
		t.Fatalf("expected warning, got %+v", sum)
	}
	if findings[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity")
	}
}

func TestValidateMermaid_ValidDiagramPasses(t *testing.T) {
	var findings []Finding
	var sum fileMetrics
	validateMermaidBlock("x.md", 1, "graph TD\nA --> B", true, &findings, &sum)
	if len(findings) != 0 || sum.MermaidFailures != 0 {
		t.Fatalf("expected clean, got %+v / %+v", findings, sum)
	}
	if sum.MermaidValidated != 1 {
		t.Fatalf("expected MermaidValidated=1")
	}
}

func TestBalancedBrackets(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"", true}, {"()", true}, {"({[]})", true}, {"(", false}, {"([)]", false},
	}
	for _, c := range cases {
		if got := balancedBrackets(c.s); got != c.want {
			t.Errorf("balancedBrackets(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

// --- absolute paths --------------------------------------------------------

func TestScanAbsolutePath_DetectsUnixPath(t *testing.T) {
	cfg := newCfg()
	var findings []Finding
	var sum fileMetrics
	scanAbsolutePath("doc.md", "see /home/user/x.txt", "see /home/user/x.txt", 3, cfg, &findings, &sum)
	if sum.AbsoluteFailures != 1 {
		t.Fatalf("expected 1 abs failure, got %+v", sum)
	}
}

func TestScanAbsolutePath_AllowlistPermits(t *testing.T) {
	cfg := newCfg()
	cfg.pathAllow = []string{"/home/"}
	var findings []Finding
	var sum fileMetrics
	scanAbsolutePath("doc.md", "see /home/user/x.txt", "", 1, cfg, &findings, &sum)
	if sum.AbsoluteFailures != 0 {
		t.Fatalf("expected allowlist to suppress failure, got %+v", sum)
	}
	if sum.AbsoluteHits != 1 {
		t.Fatalf("expected hit to be recorded, got %+v", sum)
	}
}

func TestScanAbsolutePath_WindowsPath(t *testing.T) {
	cfg := newCfg()
	var findings []Finding
	var sum fileMetrics
	scanAbsolutePath("doc.md", `C:\Users\me`, "", 1, cfg, &findings, &sum)
	if sum.AbsoluteFailures != 1 {
		t.Fatalf("expected windows path failure, got %+v", sum)
	}
}

// --- links -----------------------------------------------------------------

func TestValidateLinks_BrokenLocalLink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.md")
	writeTestFile(t, src, "")
	links := []linkTarget{{File: src, Line: 1, Dest: "missing.md", location: src + ":1"}}
	parsed, _ := url.Parse("missing.md")
	if ok, _ := validateLocalLink(dir, links[0], parsed, newCfg()); ok {
		t.Fatal("expected broken local link")
	}
}

func TestValidateLinks_ValidLocalLink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.md")
	dest := filepath.Join(dir, "b.md")
	writeTestFile(t, src, "")
	writeTestFile(t, dest, "")
	parsed, _ := url.Parse("b.md")
	if ok, _ := validateLocalLink(dir, linkTarget{File: src}, parsed, newCfg()); !ok {
		t.Fatal("expected valid local link")
	}
}

func TestCheckExternalLink_404Fails(t *testing.T) {
	doer := &fakeDoer{resp: map[string]*http.Response{
		"https://example.com/missing": {StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))},
	}}
	status, _ := checkExternalLink(context.Background(), doer, "https://example.com/missing", newCfg())
	if status != "fail" {
		t.Fatalf("expected fail, got %s", status)
	}
}

func TestCheckExternalLink_OK(t *testing.T) {
	doer := &fakeDoer{}
	status, _ := checkExternalLink(context.Background(), doer, "https://example.com/", newCfg())
	if status != "ok" {
		t.Fatalf("expected ok, got %s", status)
	}
}

func TestCheckExternalLink_LocalhostIgnored(t *testing.T) {
	doer := &fakeDoer{}
	status, _ := checkExternalLink(context.Background(), doer, "http://localhost:1234/", newCfg())
	if status != "ok" || len(doer.called) != 0 {
		t.Fatalf("expected localhost to short-circuit, status=%s called=%v", status, doer.called)
	}
}

func TestShouldIgnoreLink_PatternsAndLocalhost(t *testing.T) {
	if !shouldIgnoreLink("http://localhost:9000/x", nil) {
		t.Fatal("localhost should be ignored")
	}
	if !shouldIgnoreLink("https://internal.example.com/x", []string{"https://internal.example.com/*"}) {
		t.Fatal("pattern should match")
	}
}

// --- bidirectional refs ----------------------------------------------------

func TestValidateCodeRef_BrokenWhenMissing(t *testing.T) {
	dir := t.TempDir()
	err := validateCodeRef(dir, codeRefTarget{FilePath: "no/such/file.go"})
	if err == nil {
		t.Fatal("expected error for missing ref")
	}
}

func TestValidateCodeRef_OKWhenPresent(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "src/foo.go"), "package x\n")
	if err := validateCodeRef(dir, codeRefTarget{FilePath: "src/foo.go"}); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestValidateDocRef_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateDocRef(dir, docRefTarget{DocPath: "docs"}); err == nil {
		t.Fatal("expected directory rejection")
	}
}

func TestExtractCodeRefs_EmptyContent(t *testing.T) {
	if got := extractCodeRefs("x.md", ""); len(got) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(got))
	}
}

func TestScanCodeFilesForDocRefs_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "src/a.go"), "// DOC: docs/x.md\n")
	writeTestFile(t, filepath.Join(dir, "node_modules/lib/b.go"), "// DOC: docs/y.md\n")
	refs, _, err := scanCodeFilesForDocRefs(context.Background(), dir, newCfg())
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, ref := range refs {
		if strings.Contains(ref.File, "node_modules") {
			t.Fatalf("node_modules should be skipped, got %+v", ref)
		}
	}
}

func TestValidateBidirectionalRefs_ValidatesCLIRefsThroughCLIHealth(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "docs", "commands.md")
	writeTestFile(t, doc, strings.Join([]string{
		"Use `cli:cli-health command validate --command \"vrooli scenario test cli-health\"`.",
		"Fix `cli:knowledge-observatory docs healt cli-health`.",
		"Partial `cli:knowledge-observatory docs health cli-health --checks=refs`.",
		"Future `cli[future]:future-tool launch`.",
	}, "\n"))
	validator := &fakeCommandValidator{results: map[string]CommandReferenceResult{
		`cli-health command validate --command "vrooli scenario test cli-health"`: {
			Verdict:         "valid",
			ValidationLevel: "argument_shape_validated",
		},
		"knowledge-observatory docs healt cli-health": {
			Verdict:         "invalid",
			ValidationLevel: "owner_identified",
			Issues:          []CommandReferenceIssue{{Code: "unknown_command", Message: "command path was not found"}},
			Suggestions:     []CommandReferenceSuggestion{{Command: "knowledge-observatory docs health"}},
		},
		"knowledge-observatory docs health cli-health --checks=refs": {
			Verdict:         "partial",
			ValidationLevel: "command_exists",
			Issues:          []CommandReferenceIssue{{Code: "argument_schema_unavailable", Message: "arguments unavailable"}},
		},
	}}

	findings, summary := validateBidirectionalRefs(context.Background(), dir, []string{doc}, newCfg(), validator)

	if summary.MarkedRefsFound != 4 {
		t.Fatalf("MarkedRefsFound = %d, want 4", summary.MarkedRefsFound)
	}
	if summary.MarkedRefsBroken != 1 {
		t.Fatalf("MarkedRefsBroken = %d, want 1; findings=%#v", summary.MarkedRefsBroken, findings)
	}
	if summary.MarkedRefsSkipped != 1 {
		t.Fatalf("MarkedRefsSkipped = %d, want 1", summary.MarkedRefsSkipped)
	}
	if !hasFinding(findings, "broken_marked_ref") {
		t.Fatalf("expected broken CLI ref finding, got %#v", findings)
	}
	if !hasFinding(findings, "partial_cli_ref") {
		t.Fatalf("expected partial CLI ref finding, got %#v", findings)
	}
	if len(validator.calls) != 3 {
		t.Fatalf("validator calls = %d, want 3", len(validator.calls))
	}
}

func TestValidateCommandSnippets_ValidatesVrooliOwnedShellFences(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	scenarioDir := filepath.Join(scenariosRoot, "demo")
	for _, name := range []string{"cli-health", "knowledge-observatory"} {
		if err := os.MkdirAll(filepath.Join(scenariosRoot, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	doc := filepath.Join(scenarioDir, "docs", "commands.md")
	writeTestFile(t, doc, strings.Join([]string{
		"# Commands",
		"```bash",
		"$ cli-health command validate --command \"vrooli scenario test cli-health\"",
		"git status",
		"knowledge-observatory docs healt cli-health",
		"knowledge-observatory docs health cli-health --checks=refs",
		"```",
		"```text",
		"cli-health command typo",
		"```",
	}, "\n"))
	validator := &fakeCommandValidator{results: map[string]CommandReferenceResult{
		`cli-health command validate --command "vrooli scenario test cli-health"`: {
			Verdict:         "valid",
			ValidationLevel: "argument_shape_validated",
		},
		"knowledge-observatory docs healt cli-health": {
			Verdict:         "invalid",
			ValidationLevel: "owner_identified",
			Issues:          []CommandReferenceIssue{{Code: "unknown_command", Message: "command path was not found"}},
			Suggestions:     []CommandReferenceSuggestion{{Command: "knowledge-observatory docs health"}},
		},
		"knowledge-observatory docs health cli-health --checks=refs": {
			Verdict:         "partial",
			ValidationLevel: "command_exists",
			Issues:          []CommandReferenceIssue{{Code: "argument_schema_unavailable", Message: "arguments unavailable"}},
		},
	}}

	findings := validateCommandSnippets(context.Background(), scenarioDir, []string{doc}, validator)

	if len(validator.calls) != 3 {
		t.Fatalf("validator calls = %d, want 3 (%#v)", len(validator.calls), validator.calls)
	}
	if !hasFinding(findings, "broken_command_snippet") {
		t.Fatalf("expected broken command snippet, got %#v", findings)
	}
	if !hasFinding(findings, "partial_command_snippet") {
		t.Fatalf("expected partial command snippet, got %#v", findings)
	}
	if hasFinding(findings, "unknown_command_snippet") {
		t.Fatalf("did not expect unknown command snippet, got %#v", findings)
	}
}

// --- manifest coverage -----------------------------------------------------

func TestCheckManifestCoverage_NoManifestNoFindings(t *testing.T) {
	dir := t.TempDir()
	manifestRel := docsManifestRel(t)
	out, _, err := checkManifestCoverage(dir, manifestRel, false, nil)
	if err != nil || len(out) != 0 {
		t.Fatalf("expected empty when manifest missing, got err=%v findings=%v", err, out)
	}
}

func TestCheckManifestCoverage_MissingDocReported(t *testing.T) {
	dir := t.TempDir()
	manifestRel := docsManifestRel(t)
	writeTestFile(t, filepath.Join(dir, filepath.FromSlash(manifestRel)), `["docs/missing.md"]`)
	out, cov, err := checkManifestCoverage(dir, manifestRel, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cov.MissingDocs) != 1 || cov.MissingDocs[0] != "docs/missing.md" {
		t.Fatalf("expected missing doc, got %+v", cov)
	}
	if !hasFinding(out, "manifest_missing_doc") {
		t.Fatalf("expected manifest_missing_doc finding, got %+v", out)
	}
}

func TestCheckManifestCoverage_OrphanReportedOnlyInStrict(t *testing.T) {
	dir := t.TempDir()
	manifestRel := docsManifestRel(t)
	writeTestFile(t, filepath.Join(dir, filepath.FromSlash(manifestRel)), `[]`)
	writeTestFile(t, filepath.Join(dir, "docs/orphan.md"), "x")
	foundDocs := []string{filepath.Join(dir, "docs/orphan.md")}
	out, cov, err := checkManifestCoverage(dir, manifestRel, false, foundDocs)
	if err != nil {
		t.Fatal(err)
	}
	if cov.NotInManifest != 1 {
		t.Fatalf("expected orphan recorded, got %+v", cov)
	}
	if hasFinding(out, "manifest_orphaned_doc") {
		t.Fatal("orphan should not be a finding when requireAll=false")
	}

	out, _, err = checkManifestCoverage(dir, manifestRel, true, foundDocs)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(out, "manifest_orphaned_doc") {
		t.Fatal("orphan should be a finding when requireAll=true")
	}
}

func TestCheckManifestCoverage_ManifestObjectFormat(t *testing.T) {
	dir := t.TempDir()
	manifestRel := docsManifestRel(t)
	writeTestFile(t, filepath.Join(dir, filepath.FromSlash(manifestRel)), `{"docs":["docs/x.md"]}`)
	writeTestFile(t, filepath.Join(dir, "docs/x.md"), "x")
	_, cov, err := checkManifestCoverage(dir, manifestRel, false, []string{filepath.Join(dir, "docs/x.md")})
	if err != nil {
		t.Fatal(err)
	}
	if cov.InManifest != 1 {
		t.Fatalf("expected InManifest=1, got %+v", cov)
	}
}

func TestCheckManifestCoverage_MalformedFails(t *testing.T) {
	dir := t.TempDir()
	manifestRel := docsManifestRel(t)
	writeTestFile(t, filepath.Join(dir, filepath.FromSlash(manifestRel)), `{not json`)
	_, _, err := checkManifestCoverage(dir, manifestRel, false, nil)
	if err == nil {
		t.Fatal("expected malformed-manifest error")
	}
}

// --- walk / exclude --------------------------------------------------------

func TestCollectMarkdownFiles_SkipsBuildDirs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "docs/a.md"), "x")
	writeTestFile(t, filepath.Join(dir, "node_modules/lib/b.md"), "x")
	files, err := collectMarkdownFiles(dir, newCfg())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(f, "node_modules") {
			t.Fatalf("node_modules should be skipped: %s", f)
		}
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %v", files)
	}
}

func TestShouldExcludePath_GlobMatching(t *testing.T) {
	cfg := newCfg()
	cfg.scanExcludeGlobs = []string{"archived/**"}
	dir := "/tmp/foo"
	if !shouldExcludePath(dir, "/tmp/foo/archived/old.md", false, cfg) {
		t.Fatal("expected archived/** glob to match")
	}
	if shouldExcludePath(dir, "/tmp/foo/docs/new.md", false, cfg) {
		t.Fatal("docs/new.md should not match archived/**")
	}
}

func TestAllowedPrefix(t *testing.T) {
	if allowedPrefix("/etc/x", nil) {
		t.Fatal("empty allow list should not allow anything")
	}
	if !allowedPrefix("/api/v1/x", []string{"/api/"}) {
		t.Fatal("expected /api/ prefix match")
	}
	if allowedPrefix("/etc/x", []string{"/api/"}) {
		t.Fatal("/etc should not match /api")
	}
}

// --- end-to-end DocHealth on a small scenario ------------------------------

func TestService_DocHealth_BasicScenario(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	scenarioDir := filepath.Join(scenarios, "demo")
	writeTestFile(t, filepath.Join(scenarioDir, "docs/README.md"), "# Demo\n\nsee [other](other.md)\n")
	writeTestFile(t, filepath.Join(scenarioDir, "docs/other.md"), "ok")

	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	res, err := svc.DocHealth(context.Background(), "demo", DocHealthOptions{})
	if err != nil {
		t.Fatalf("DocHealth failed: %v", err)
	}
	if res.Counts.FilesChecked < 2 {
		t.Fatalf("expected at least 2 files checked, got %d", res.Counts.FilesChecked)
	}
	if res.Counts.BrokenLinks != 0 {
		t.Fatalf("expected no broken links, got %d", res.Counts.BrokenLinks)
	}
}

func TestService_DocHealth_RejectsBadName(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenarios, 0o755); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DocHealth(context.Background(), "..", DocHealthOptions{}); err == nil {
		t.Fatal("expected name rejection")
	}
	if _, err := svc.DocHealth(context.Background(), "no/slash", DocHealthOptions{}); err == nil {
		t.Fatal("expected slash rejection")
	}
}

// --- shared helpers --------------------------------------------------------

func hasFinding(fs []Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}
