package docs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

// Runner executes docs validations.
type Runner struct {
	config   Config
	settings *Settings
	log      io.Writer
	client   *http.Client
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
	return r
}

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

	success := summary.MarkdownFailures == 0 &&
		summary.MermaidFailures == 0 &&
		summary.BrokenLinks == 0 &&
		summary.AbsoluteFailures == 0 &&
		len(allErrors) == 0

	summaryLine := fmt.Sprintf(
		"Docs summary: files=%d mermaid=%d validated (%d failures) markdown(warn=%d, fail=%d) links(local=%d external=%d broken=%d) absolute(hits=%d blocked=%d)",
		summary.FilesChecked,
		summary.MermaidValidated, summary.MermaidFailures,
		summary.MarkdownWarnings, summary.MarkdownFailures,
		summary.LocalLinks, summary.ExternalLinks, summary.BrokenLinks,
		summary.AbsolutePathHits, summary.AbsoluteFailures,
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
	absUnixPathPattern    = regexp.MustCompile(`/(Users|home|var|etc|opt|srv|private|Volumes)/`)
	absWindowsPathPattern = regexp.MustCompile(`^[A-Za-z]:\\`)
	mermaidHeaderPattern  = regexp.MustCompile(`^(graph|flowchart|flowchart\s+(TB|TD|LR|RL)|sequenceDiagram|classDiagram|stateDiagram|stateDiagram-v2|gantt|journey|erDiagram|pie)\b`)
)

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
			if allowedPrefix(dest, r.settings.Paths.Allow) {
				return true
			}
			return false
		}
		// Root-relative site paths are treated as portable by default.
		return true
	}

	base := filepath.Dir(link.File)
	target := dest
	if strings.HasPrefix(dest, "#") {
		target = ""
	}

	if target != "" {
		target = strings.TrimPrefix(target, "./")
		target = filepath.Clean(filepath.Join(base, target))
		info, err := os.Stat(target)
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func (r *Runner) checkExternalLink(ctx context.Context, target string) (string, error) {
	if r.shouldIgnoreLink(target) {
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
