package docs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"test-genie/internal/shared"
)

// Config controls docs validation.
type Config struct {
	ScenarioDir  string
	ScenarioName string
	Settings     *Settings
	HTTPClient   *http.Client
}

// LinkIgnoreChecker determines if a URL should be skipped during external link validation.
// This is a seam that allows tests to bypass the default localhost detection.
type LinkIgnoreChecker func(url string) bool

// Runner executes docs validations.
type Runner struct {
	config        Config
	settings      *Settings
	log           io.Writer
	client        *http.Client
	ignoreChecker LinkIgnoreChecker
}

// Option configures a Runner.
type Option func(*Runner)

// WithLogger sets a log writer.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) { r.log = w }
}

// WithHTTPClient overrides the HTTP client used for external link checks.
func WithHTTPClient(client *http.Client) Option {
	return func(r *Runner) { r.client = client }
}

// WithLinkIgnoreChecker overrides the function that determines which URLs to skip.
// This is primarily useful for testing external link validation without the
// default localhost/127.0.0.1 bypass.
func WithLinkIgnoreChecker(checker LinkIgnoreChecker) Option {
	return func(r *Runner) { r.ignoreChecker = checker }
}

// New creates a Runner.
func New(config Config, opts ...Option) *Runner {
	settings := config.Settings
	if settings == nil {
		settings = DefaultSettings()
	}

	r := &Runner{
		config:   config,
		settings: settings,
		log:      io.Discard,
		client: &http.Client{
			Timeout: settings.linksTimeout(),
		},
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.client == nil {
		r.client = &http.Client{Timeout: settings.linksTimeout()}
	}
	// Set default ignore checker after options so it can access settings
	if r.ignoreChecker == nil {
		r.ignoreChecker = r.shouldIgnoreLink
	}
	return r
}

// resolvePath resolves a target path relative to the scenario directory or a base file.
// If base is empty, resolves relative to scenario root.
// If base is a file path, resolves relative to that file's directory.
func (r *Runner) resolvePath(target, base string) string {
	if filepath.IsAbs(target) {
		return target
	}
	if base == "" {
		return filepath.Join(r.config.ScenarioDir, target)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(base), target))
}

// DOC: docs/phases/docs/README.md#validation-flow
// Run executes docs validation and returns the aggregated result.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	shared.LogInfo(r.log, "Starting docs validation for %s", r.config.ScenarioName)

	files, err := r.collectMarkdownFiles()
	if err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Ensure docs files are readable",
		}
	}
	if len(files) == 0 {
		shared.LogInfo(r.log, "Docs scan: no Markdown files found")
		return &RunResult{
			Success: true,
			Observations: []Observation{
				NewInfoObservation("No Markdown files found"),
			},
			Summary: Summary{},
		}
	}
	shared.LogInfo(r.log, "Docs scan: %d Markdown file(s) detected", len(files))

	var (
		obs       []Observation
		summary   Summary
		allErrors []string
	)

	linkTasks := make([]linkTarget, 0)

	for _, file := range files {
		fileObs, fileSummary, fileLinks, fileErrors := r.inspectFile(file)
		obs = append(obs, fileObs...)
		summary.FilesChecked++
		summary.MermaidValidated += fileSummary.MermaidValidated
		summary.MermaidFailures += fileSummary.MermaidFailures
		summary.MarkdownWarnings += fileSummary.MarkdownWarnings
		summary.MarkdownFailures += fileSummary.MarkdownFailures
		summary.AbsoluteFailures += fileSummary.AbsoluteFailures
		summary.AbsolutePathHits += fileSummary.AbsoluteHits
		if len(fileLinks) > 0 {
			linkTasks = append(linkTasks, fileLinks...)
		}
		allErrors = append(allErrors, fileErrors...)
	}

	// Process links after scanning all files so we can deduplicate and parallelize.
	linkObs, linkSummary := r.validateLinks(ctx, linkTasks)
	obs = append(obs, linkObs...)
	summary.LocalLinks += linkSummary.LocalLinks
	summary.ExternalLinks += linkSummary.ExternalLinks
	summary.BrokenLinks += linkSummary.BrokenLinks
	summary.ExternalWarnings += linkSummary.ExternalWarnings
	summary.ExternalFailures += linkSummary.ExternalFailures

	// Bidirectional reference validation
	if r.settings.referencesEnabled() {
		refObs, refSummary := r.validateBidirectionalRefs(ctx, files)
		obs = append(obs, refObs...)
		summary.CodeRefsFound = refSummary.CodeRefsFound
		summary.CodeRefsBroken = refSummary.CodeRefsBroken
		summary.DocRefsFound = refSummary.DocRefsFound
		summary.DocRefsBroken = refSummary.DocRefsBroken
		summary.CodeFilesScanned = refSummary.CodeFilesScanned
	}

	// Manifest coverage check
	if r.settings.manifestEnabled() {
		coverage, err := r.checkManifestCoverage(files)
		if err != nil {
			shared.LogInfo(r.log, "Warning: manifest check failed: %v", err)
		} else {
			summary.DocsInManifest = coverage.InManifest
			summary.DocsNotInManifest = coverage.NotInManifest

			for _, missing := range coverage.MissingDocs {
				obs = append(obs, NewWarningObservation(fmt.Sprintf("manifest references missing doc: %s", missing)))
			}
			if r.settings.manifestRequireAll() {
				for _, orphan := range coverage.OrphanedDocs {
					obs = append(obs, NewWarningObservation(fmt.Sprintf("doc not in manifest: %s", orphan)))
				}
			}
		}
	}

	// Calculate reference failures based on strict mode
	referenceFailures := 0
	if r.settings.referencesEnabled() && r.settings.referencesStrict() {
		referenceFailures = summary.CodeRefsBroken + summary.DocRefsBroken
	}

	success := summary.MarkdownFailures == 0 &&
		summary.MermaidFailures == 0 &&
		summary.BrokenLinks == 0 &&
		summary.AbsoluteFailures == 0 &&
		referenceFailures == 0 &&
		len(allErrors) == 0

	summaryLine := fmt.Sprintf(
		"Docs summary: files=%d mermaid=%d validated (%d failures) markdown(warn=%d, fail=%d) links(local=%d external=%d broken=%d) absolute(hits=%d blocked=%d) refs(code=%d/%d doc=%d/%d)",
		summary.FilesChecked,
		summary.MermaidValidated, summary.MermaidFailures,
		summary.MarkdownWarnings, summary.MarkdownFailures,
		summary.LocalLinks, summary.ExternalLinks, summary.BrokenLinks,
		summary.AbsolutePathHits, summary.AbsoluteFailures,
		summary.CodeRefsFound, summary.CodeRefsBroken,
		summary.DocRefsFound, summary.DocRefsBroken,
	)
	shared.LogInfo(r.log, "%s", summaryLine)
	fmt.Fprintln(r.log, summaryLine)

	if success {
		obs = append(obs, NewSuccessObservation(fmt.Sprintf("Docs validation passed (%s)", summary.String())))
		return &RunResult{
			Success:      true,
			Observations: obs,
			Summary:      summary,
		}
	}

	msg := "Docs validation failed"
	if len(allErrors) > 0 {
		msg = fmt.Sprintf("%s: %s", msg, strings.Join(allErrors, "; "))
	}
	return &RunResult{
		Success:      false,
		Error:        errors.New(msg),
		FailureClass: FailureClassMisconfiguration,
		Remediation:  "Fix docs issues and re-run",
		Observations: obs,
		Summary:      summary,
	}
}

type fileSummary struct {
	MermaidValidated int
	MermaidFailures  int
	MarkdownWarnings int
	MarkdownFailures int
	AbsoluteFailures int
	AbsoluteHits     int
}

type linkTarget struct {
	File     string
	Line     int
	Dest     string
	isImage  bool
	location string
}

var (
	markdownLinkPattern   = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
	codeFencePattern      = regexp.MustCompile("^(```|~~~)([a-zA-Z0-9_-]+)?")
	inlineCodePattern     = regexp.MustCompile("`[^`]*`")
	absUnixPathPattern    = regexp.MustCompile(`/(Users|home|var|etc|opt|srv|private|Volumes)/`)
	absWindowsPathPattern = regexp.MustCompile(`^[A-Za-z]:\\`)
	mermaidHeaderPattern  = regexp.MustCompile(`^(graph|flowchart|flowchart\s+(TB|TD|LR|RL)|sequenceDiagram|classDiagram|stateDiagram|stateDiagram-v2|gantt|journey|erDiagram|pie)\b`)

	// Bidirectional reference patterns
	// Matches [CODE: path/to/file.go] or [CODE: path/to/file.go#FunctionName] or [CODE: path/to/file.go:42]
	codeRefPattern = regexp.MustCompile(`\[CODE:\s*([^\]]+)\]`)
	// Matches standalone // DOC: path/to/doc.md or /* DOC: path/to/doc.md */ or # DOC: path/to/doc.md
	docRefPattern = regexp.MustCompile(`^\s*(?://|/\*|#)\s*DOC:\s*([^\s\*\n]+)`)
)

// codeRefTarget represents a [CODE: ...] reference found in documentation.
type codeRefTarget struct {
	File     string // The markdown file containing the reference
	Ref      string // The raw reference string
	FilePath string // Extracted file path (without anchor/line number)
	Line     int    // Line number in the markdown file
}

// docRefTarget represents a // DOC: comment found in code.
type docRefTarget struct {
	File    string // The code file containing the comment
	Ref     string // The raw reference string
	DocPath string // Extracted doc path (without anchor)
	Line    int    // Line number in the code file
}

func (r *Runner) collectMarkdownFiles() ([]string, error) {
	var files []string
	skipDirs := map[string]struct{}{
		".git": {}, "node_modules": {}, "dist": {}, "build": {}, ".turbo": {}, ".next": {},
		".pnpm-store": {}, "coverage": {}, ".cache": {}, "tmp": {}, "logs": {},
	}
	err := filepath.WalkDir(r.config.ScenarioDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if r.shouldExcludePath(path, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") || strings.HasSuffix(d.Name(), ".mdx") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (r *Runner) inspectFile(path string) ([]Observation, fileSummary, []linkTarget, []string) {
	var (
		obs     []Observation
		summary fileSummary
		links   []linkTarget
		errors  []string
	)

	if !r.settings.markdownEnabled() {
		obs = append(obs, NewSkipObservation(fmt.Sprintf("%s: markdown validation disabled", path)))
		return obs, summary, links, errors
	}

	f, err := os.Open(path)
	if err != nil {
		errors = append(errors, fmt.Sprintf("cannot read %s: %v", path, err))
		return obs, summary, links, errors
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	inFence := false
	fenceLang := ""
	fenceMarker := ""
	var mermaidBuf []string
	var mermaidStart int

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trim := strings.TrimSpace(line)

		// Detect code fences for markdown syntax sanity.
		if matches := codeFencePattern.FindStringSubmatch(trim); len(matches) > 0 {
			marker := matches[1]
			lang := strings.TrimSpace(matches[2])
			if !inFence {
				inFence = true
				fenceMarker = marker
				fenceLang = lang
				mermaidBuf = mermaidBuf[:0]
				mermaidStart = lineNum
			} else if marker == fenceMarker {
				// Closing fence
				if fenceLang == "mermaid" || fenceLang == "mermaidjs" {
					if r.settings.mermaidEnabled() {
						r.validateMermaidBlock(path, mermaidStart, strings.Join(mermaidBuf, "\n"), &obs, &summary)
					} else {
						obs = append(obs, NewSkipObservation(fmt.Sprintf("%s:%d mermaid validation disabled", path, mermaidStart)))
					}
				}
				inFence = false
				fenceLang = ""
				fenceMarker = ""
				mermaidBuf = mermaidBuf[:0]
			}
			continue
		}

		if inFence {
			if fenceLang == "mermaid" || fenceLang == "mermaidjs" {
				mermaidBuf = append(mermaidBuf, line)
			}
			continue
		}

		// Link extraction outside code fences.
		for _, match := range markdownLinkPattern.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 4 {
				continue
			}
			start, end := match[2], match[3]
			dest := strings.TrimSpace(line[start:end])
			if dest == "" {
				continue
			}
			// Strip angle brackets
			dest = strings.Trim(dest, "<>")
			isImage := line[match[0]] == '!'
			links = append(links, linkTarget{
				File:     path,
				Line:     lineNum,
				Dest:     dest,
				isImage:  isImage,
				location: fmt.Sprintf("%s:%d", path, lineNum),
			})
		}

		// Absolute path detection
		if r.settings.pathsEnabled() {
			var absMatch string
			switch {
			case absUnixPathPattern.MatchString(line):
				absMatch = absUnixPathPattern.FindString(line)
			case absWindowsPathPattern.MatchString(line):
				absMatch = absWindowsPathPattern.FindString(line)
			}
			if absMatch != "" {
				summary.AbsoluteHits++
				if allowedPrefix(absMatch, r.settings.Paths.Allow) {
					continue
				}
				summary.AbsoluteFailures++
				obs = append(obs, NewErrorObservation(fmt.Sprintf("%s:%d contains absolute filesystem path", path, lineNum)))
				fmt.Fprintf(r.log, "ABSOLUTE_PATH %s:%d -> %s\n", path, lineNum, trim)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		errors = append(errors, fmt.Sprintf("failed to read %s: %v", path, err))
	}

	if inFence {
		summary.MarkdownFailures++
		msg := fmt.Sprintf("%s:%d code fence not closed", path, mermaidStart)
		obs = append(obs, NewErrorObservation(msg))
	}

	return obs, summary, links, errors
}

func (r *Runner) validateMermaidBlock(file string, line int, content string, obs *[]Observation, summary *fileSummary) {
	summary.MermaidValidated++
	if mermaidHeaderPattern.MatchString(strings.TrimSpace(content)) && balancedBrackets(content) {
		return
	}

	message := fmt.Sprintf("%s:%d mermaid diagram appears invalid", file, line)
	if r.settings.mermaidStrict() {
		summary.MermaidFailures++
		*obs = append(*obs, NewErrorObservation(message))
	} else {
		summary.MarkdownWarnings++
		*obs = append(*obs, NewWarningObservation(message))
	}
}

func balancedBrackets(content string) bool {
	var stack []rune
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
	for _, r := range content {
		switch r {
		case '(', '[', '{':
			stack = append(stack, r)
		case ')', ']', '}':
			if len(stack) == 0 || stack[len(stack)-1] != pairs[r] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return len(stack) == 0
}

type linkSummary struct {
	LocalLinks       int
	ExternalLinks    int
	BrokenLinks      int
	ExternalWarnings int
	ExternalFailures int
}

func (r *Runner) validateLinks(ctx context.Context, links []linkTarget) ([]Observation, linkSummary) {
	if len(links) == 0 || !r.settings.linksEnabled() {
		return nil, linkSummary{}
	}

	var (
		obs     []Observation
		summary linkSummary
	)

	seen := make(map[string]struct{})
	var external []linkTarget
	for _, link := range links {
		parsed, err := url.Parse(link.Dest)
		if err != nil {
			summary.BrokenLinks++
			obs = append(obs, NewErrorObservation(fmt.Sprintf("%s invalid link target '%s': %v", link.location, link.Dest, err)))
			continue
		}
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			key := parsed.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			external = append(external, link)
			continue
		}

		// local link
		summary.LocalLinks++
		if ok := r.validateLocalLink(link, parsed); !ok {
			summary.BrokenLinks++
			obs = append(obs, NewErrorObservation(fmt.Sprintf("%s broken local link '%s'", link.location, link.Dest)))
		}
	}

	if len(external) == 0 {
		return obs, summary
	}

	// External link checking with concurrency.
	concurrency := r.settings.linksConcurrency()
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, link := range external {
		link := link
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ctxReq, cancel := context.WithTimeout(ctx, r.settings.linksTimeout())
			defer cancel()

			status, err := r.checkExternalLink(ctxReq, link.Dest)
			mu.Lock()
			summary.ExternalLinks++
			switch status {
			case "ok":
			case "warn":
				summary.ExternalWarnings++
				obs = append(obs, NewWarningObservation(fmt.Sprintf("%s external link warning for '%s': %v", link.location, link.Dest, err)))
			case "fail":
				summary.ExternalFailures++
				summary.BrokenLinks++
				obs = append(obs, NewErrorObservation(fmt.Sprintf("%s broken external link '%s': %v", link.location, link.Dest, err)))
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	return obs, summary
}

func (r *Runner) validateLocalLink(link linkTarget, parsed *url.URL) bool {
	dest := parsed.Path
	if dest == "" || dest == "#" {
		return true
	}

	// guard against absolute paths
	if r.settings.pathsEnabled() && strings.HasPrefix(dest, "/") {
		// Treat OS-rooted paths as failures unless explicitly allowed
		if absUnixPathPattern.MatchString(dest) || absWindowsPathPattern.MatchString(dest) {
			return allowedPrefix(dest, r.settings.Paths.Allow)
		}
		// Root-relative site paths are treated as portable by default.
		return true
	}

	target := dest
	if strings.HasPrefix(dest, "#") {
		target = ""
	}

	if target != "" {
		target = strings.TrimPrefix(target, "./")
		target = r.resolvePath(target, link.File)
		info, err := os.Stat(target)
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func (r *Runner) checkExternalLink(ctx context.Context, target string) (string, error) {
	if r.ignoreChecker(target) {
		return "ok", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	if err != nil {
		return "fail", err
	}
	resp, err := r.client.Do(req)
	if err != nil || resp.StatusCode >= http.StatusBadRequest {
		// fallback to GET if HEAD blocked
		req.Method = http.MethodGet
		resp, err = r.client.Do(req)
	}
	if err != nil {
		if r.settings.linksStrictExternal() {
			return "fail", err
		}
		return "warn", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return "fail", fmt.Errorf("status %d", resp.StatusCode)
	}
	return "ok", nil
}

func (r *Runner) shouldIgnoreLink(target string) bool {
	for _, pattern := range r.settings.Links.Ignore {
		if pattern == "" {
			continue
		}
		if matchPattern(pattern, target) {
			return true
		}
	}
	return strings.Contains(target, "localhost") || strings.Contains(target, "127.0.0.1")
}

// refSummary tracks bidirectional reference validation metrics.
type refSummary struct {
	CodeRefsFound    int
	CodeRefsBroken   int
	DocRefsFound     int
	DocRefsBroken    int
	CodeFilesScanned int
}

// DOC: docs/phases/docs/README.md#bidirectional-reference-validation
// validateBidirectionalRefs validates both [CODE: ...] references in docs and // DOC: comments in code.
func (r *Runner) validateBidirectionalRefs(ctx context.Context, markdownFiles []string) ([]Observation, refSummary) {
	var obs []Observation
	var summary refSummary

	// Validate [CODE: ...] references in markdown files
	if r.settings.codeRefsEnabled() {
		for _, file := range markdownFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			refs := extractCodeRefs(file, string(content))
			summary.CodeRefsFound += len(refs)

			for _, ref := range refs {
				if err := r.validateCodeRef(ref); err != nil {
					summary.CodeRefsBroken++
					location := fmt.Sprintf("%s:%d", ref.File, ref.Line)
					if r.settings.referencesStrict() {
						obs = append(obs, NewErrorObservation(fmt.Sprintf("%s broken code reference [CODE: %s]: %v", location, ref.Ref, err)))
					} else {
						obs = append(obs, NewWarningObservation(fmt.Sprintf("%s broken code reference [CODE: %s]: %v", location, ref.Ref, err)))
					}
				}
			}
		}
	}

	// Validate // DOC: comments in code files
	if r.settings.docRefsEnabled() {
		docRefs, filesScanned, err := r.scanCodeFilesForDocRefs(ctx)
		if err != nil && ctx.Err() == nil {
			shared.LogInfo(r.log, "Warning: code file scan failed: %v", err)
		}
		summary.CodeFilesScanned = filesScanned
		summary.DocRefsFound = len(docRefs)

		for _, ref := range docRefs {
			if err := r.validateDocRef(ref); err != nil {
				summary.DocRefsBroken++
				location := fmt.Sprintf("%s:%d", ref.File, ref.Line)
				if r.settings.referencesStrict() {
					obs = append(obs, NewErrorObservation(fmt.Sprintf("%s broken doc reference // DOC: %s: %v", location, ref.Ref, err)))
				} else {
					obs = append(obs, NewWarningObservation(fmt.Sprintf("%s broken doc reference // DOC: %s: %v", location, ref.Ref, err)))
				}
			}
		}
	}

	return obs, summary
}

func allowedPrefix(path string, allow []string) bool {
	if len(allow) == 0 {
		return false
	}
	for _, prefix := range allow {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func matchPattern(pattern, value string) bool {
	if strings.Contains(pattern, "*") {
		ok, _ := filepath.Match(pattern, value)
		return ok
	}
	return strings.HasPrefix(value, pattern)
}

func (r *Runner) shouldExcludePath(targetPath string, isDir bool) bool {
	rel, err := filepath.Rel(r.config.ScenarioDir, targetPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return false
	}

	for _, excluded := range r.settings.scanExcludeDirs() {
		excluded = filepath.ToSlash(strings.TrimSpace(excluded))
		excluded = strings.Trim(excluded, "/")
		if excluded == "" {
			continue
		}
		// If the configured value looks like a path, apply prefix matching.
		if strings.Contains(excluded, "/") {
			if rel == excluded || strings.HasPrefix(rel, excluded+"/") {
				return true
			}
			continue
		}
		// Otherwise treat it as a directory name match on any segment.
		for _, seg := range strings.Split(rel, "/") {
			if seg == excluded {
				return true
			}
		}
	}

	for _, glob := range r.settings.scanExcludeGlobs() {
		glob = filepath.ToSlash(strings.TrimSpace(glob))
		if glob == "" {
			continue
		}
		if doublestarMatch(glob, rel) {
			return true
		}
		// For directory checks, also allow glob to match the directory prefix.
		if isDir && doublestarMatch(glob, rel+"/") {
			return true
		}
	}

	return false
}

func doublestarMatch(glob, value string) bool {
	quoted := regexp.QuoteMeta(glob)
	quoted = strings.ReplaceAll(quoted, `\*\*`, "<<<DOUBLESTAR>>>")
	quoted = strings.ReplaceAll(quoted, `\*`, `[^/]*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `[^/]`)
	quoted = strings.ReplaceAll(quoted, "<<<DOUBLESTAR>>>", ".*")
	re, err := regexp.Compile("^" + quoted + "$")
	if err != nil {
		ok, _ := path.Match(glob, value)
		return ok
	}
	return re.MatchString(value)
}

// extractCodeRefs extracts [CODE: ...] references from markdown content.
func extractCodeRefs(file, content string) []codeRefTarget {
	var refs []codeRefTarget
	lines := strings.Split(content, "\n")
	inFence := false
	fenceMarker := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceMatch := codeFencePattern.FindStringSubmatch(trimmed); fenceMatch != nil {
			marker := fenceMatch[1]
			if inFence && marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			} else if !inFence {
				inFence = true
				fenceMarker = marker
			}
			continue
		}
		if inFence {
			continue
		}

		searchLine := inlineCodePattern.ReplaceAllString(line, "")
		for _, match := range codeRefPattern.FindAllStringSubmatch(searchLine, -1) {
			if len(match) < 2 {
				continue
			}
			rawRef := strings.TrimSpace(match[1])
			filePath := extractFilePath(rawRef)
			refs = append(refs, codeRefTarget{
				File:     file,
				Ref:      rawRef,
				FilePath: filePath,
				Line:     i + 1,
			})
		}
	}
	return refs
}

// extractFilePath extracts the file path from a reference, handling path#func and path:line formats.
func extractFilePath(ref string) string {
	// Strip anchor (#section or #FunctionName)
	if idx := strings.Index(ref, "#"); idx != -1 {
		ref = ref[:idx]
	}
	// Strip line number (:42)
	if idx := strings.LastIndex(ref, ":"); idx != -1 {
		// Make sure it's actually a line number (digits after colon)
		suffix := ref[idx+1:]
		if len(suffix) > 0 && isDigits(suffix) {
			ref = ref[:idx]
		}
	}
	return strings.TrimSpace(ref)
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// extractDocRefsFromFile reads a code file and extracts DOC: comments.
func extractDocRefsFromFile(path string) ([]docRefTarget, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var refs []docRefTarget
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		for _, match := range docRefPattern.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			rawRef := strings.TrimSpace(match[1])
			// Strip anchor if present
			docPath := rawRef
			if idx := strings.Index(docPath, "#"); idx != -1 {
				docPath = docPath[:idx]
			}
			refs = append(refs, docRefTarget{
				File:    path,
				Ref:     rawRef,
				DocPath: docPath,
				Line:    i + 1,
			})
		}
	}
	return refs, nil
}

// validateCodeRef checks if the referenced code file exists.
func (r *Runner) validateCodeRef(ref codeRefTarget) error {
	target := r.resolvePath(ref.FilePath, "")

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("file not found: %s", ref.FilePath)
	}
	if info.IsDir() {
		return fmt.Errorf("reference points to directory, not file: %s", ref.FilePath)
	}
	return nil
}

// validateDocRef checks if the referenced documentation file exists.
func (r *Runner) validateDocRef(ref docRefTarget) error {
	target := r.resolvePath(ref.DocPath, "")

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("doc not found: %s", ref.DocPath)
	}
	if info.IsDir() {
		return fmt.Errorf("reference points to directory, not file: %s", ref.DocPath)
	}
	return nil
}

// Hard-coded directories to always skip when scanning for DOC: comments.
var defaultSkipDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	".turbo":       {},
	".next":        {},
	".pnpm-store":  {},
	"coverage":     {},
	".cache":       {},
	"tmp":          {},
	"logs":         {},
	"vendor":       {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
	"target":       {},
}

// scanCodeFilesForDocRefs walks the scenario directory and extracts DOC: references from code files.
func (r *Runner) scanCodeFilesForDocRefs(ctx context.Context) ([]docRefTarget, int, error) {
	var refs []docRefTarget
	var filesScanned int

	extensions := make(map[string]struct{})
	for _, ext := range r.settings.codeExtensions() {
		extensions[ext] = struct{}{}
	}

	customSkipDirs := make(map[string]struct{})
	for _, dir := range r.settings.referencesSkipDirs() {
		customSkipDirs[dir] = struct{}{}
	}

	err := filepath.WalkDir(r.config.ScenarioDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check for context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if r.shouldExcludePath(path, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if _, skip := defaultSkipDirs[name]; skip {
				return filepath.SkipDir
			}
			if _, skip := customSkipDirs[name]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		if _, ok := extensions[ext]; !ok {
			return nil
		}

		filesScanned++
		fileRefs, err := extractDocRefsFromFile(path)
		if err != nil {
			// Log but don't fail on unreadable files
			shared.LogInfo(r.log, "Warning: could not read %s: %v", path, err)
			return nil
		}
		refs = append(refs, fileRefs...)
		return nil
	})

	return refs, filesScanned, err
}

// manifestCoverage tracks which docs are in/out of the manifest.
type manifestCoverage struct {
	InManifest    int
	NotInManifest int
	MissingDocs   []string // docs in manifest but not on disk
	OrphanedDocs  []string // docs on disk but not in manifest
}

// checkManifestCoverage reads the manifest and compares against found docs.
func (r *Runner) checkManifestCoverage(foundDocs []string) (manifestCoverage, error) {
	var coverage manifestCoverage

	manifestPath := filepath.Join(r.config.ScenarioDir, r.settings.manifestPath())
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No manifest file - nothing to check
			return coverage, nil
		}
		return coverage, err
	}

	// Parse manifest - expect a JSON array of paths or an object with a "docs" array
	var manifestDocs []string

	// Try parsing as simple array first
	if err := parseJSONArray(data, &manifestDocs); err != nil {
		// Try parsing as object with "docs" key
		if err := parseJSONDocsField(data, &manifestDocs); err != nil {
			return coverage, fmt.Errorf("invalid manifest format: %v", err)
		}
	}

	// Build set of manifest docs
	manifestSet := make(map[string]struct{})
	for _, doc := range manifestDocs {
		manifestSet[doc] = struct{}{}
	}

	// Build set of found docs (relative to scenario dir)
	foundSet := make(map[string]struct{})
	for _, doc := range foundDocs {
		rel, err := filepath.Rel(r.config.ScenarioDir, doc)
		if err != nil {
			rel = doc
		}
		foundSet[rel] = struct{}{}
	}

	// Check coverage
	for doc := range manifestSet {
		fullPath := filepath.Join(r.config.ScenarioDir, doc)
		if _, err := os.Stat(fullPath); err != nil {
			coverage.MissingDocs = append(coverage.MissingDocs, doc)
		} else {
			coverage.InManifest++
		}
	}

	for doc := range foundSet {
		if _, ok := manifestSet[doc]; !ok {
			coverage.OrphanedDocs = append(coverage.OrphanedDocs, doc)
			coverage.NotInManifest++
		}
	}

	return coverage, nil
}

// parseJSONArray attempts to parse data as a JSON array of strings.
func parseJSONArray(data []byte, out *[]string) error {
	// Simple check - if it starts with [, try to parse as array
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, "[") {
		return fmt.Errorf("not a JSON array")
	}

	// Use encoding/json for proper parsing
	return json.Unmarshal(data, out)
}

// parseJSONDocsField attempts to parse data as a JSON object with a "docs" field.
func parseJSONDocsField(data []byte, out *[]string) error {
	var obj struct {
		Docs []string `json:"docs"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*out = obj.Docs
	return nil
}
