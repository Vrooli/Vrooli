package dochealth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type linkSummary struct {
	LocalLinks       int
	ExternalLinks    int
	BrokenLinks      int
	ExternalWarnings int
	ExternalFailures int
}

// validateLinks resolves every link target discovered during file inspection.
// Local links are checked against the scenario filesystem; external links are
// probed via the injected Doer (or http.DefaultClient if nil).
func validateLinks(ctx context.Context, scenarioDir string, doer Doer, cfg effective, links []linkTarget) ([]Finding, linkSummary) {
	if len(links) == 0 {
		return nil, linkSummary{}
	}

	var (
		out     []Finding
		summary linkSummary
	)

	seen := make(map[string]struct{})
	var external []linkTarget
	for _, link := range links {
		parsed, err := url.Parse(link.Dest)
		if err != nil {
			summary.BrokenLinks++
			out = append(out, Finding{
				Code:     "broken_link_parse",
				Severity: SeverityFailure,
				Message:  fmt.Sprintf("%s invalid link target '%s': %v", link.location, link.Dest, err),
				Path:     link.File,
				Line:     link.Line,
				Target:   link.Dest,
			})
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

		summary.LocalLinks++
		if ok, detail := validateLocalLink(scenarioDir, link, parsed, cfg); !ok {
			summary.BrokenLinks++
			message := fmt.Sprintf("%s broken local link '%s'", link.location, link.Dest)
			if detail != "" {
				message = fmt.Sprintf("%s (%s)", message, detail)
			}
			out = append(out, Finding{
				Code:     "broken_local_link",
				Severity: SeverityFailure,
				Message:  message,
				Path:     link.File,
				Line:     link.Line,
				Target:   link.Dest,
			})
		}
	}

	if len(external) == 0 || cfg.skipExternal {
		return out, summary
	}

	if doer == nil {
		doer = defaultDoer(cfg.linkTimeout)
	}

	concurrency := cfg.linkConcurrency
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
			ctxReq, cancel := context.WithTimeout(ctx, cfg.linkTimeout)
			defer cancel()

			status, err := checkExternalLink(ctxReq, doer, link.Dest, cfg)
			mu.Lock()
			summary.ExternalLinks++
			switch status {
			case "ok":
			case "warn":
				summary.ExternalWarnings++
				out = append(out, Finding{
					Code:     "external_link_warning",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s external link warning for '%s': %v", link.location, link.Dest, err),
					Path:     link.File,
					Line:     link.Line,
					Target:   link.Dest,
				})
			case "fail":
				summary.ExternalFailures++
				summary.BrokenLinks++
				out = append(out, Finding{
					Code:     "broken_external_link",
					Severity: SeverityFailure,
					Message:  fmt.Sprintf("%s broken external link '%s': %v", link.location, link.Dest, err),
					Path:     link.File,
					Line:     link.Line,
					Target:   link.Dest,
				})
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	return out, summary
}

func validateLocalLink(scenarioDir string, link linkTarget, parsed *url.URL, cfg effective) (bool, string) {
	dest := parsed.Path
	if dest == "" || dest == "#" {
		return true, ""
	}

	if strings.HasPrefix(dest, "/") {
		if absUnixPathPattern.MatchString(dest) || absWindowsPathPattern.MatchString(dest) {
			return allowedPrefix(dest, cfg.pathAllow), ""
		}
		// Root-relative site paths are treated as portable by default.
		return true, ""
	}

	target := dest
	if strings.HasPrefix(dest, "#") {
		target = ""
	}
	if target != "" {
		target = strings.TrimPrefix(target, "./")
		resolved := filepath.Clean(filepath.Join(filepath.Dir(link.File), target))
		if _, err := os.Stat(resolved); err != nil {
			rel, relErr := filepath.Rel(scenarioDir, resolved)
			if relErr == nil && strings.HasPrefix(rel, "..") {
				return false, fmt.Sprintf("logical target escapes scenario root: %s", rel)
			}
			return false, ""
		}
	}
	return true, ""
}

func checkExternalLink(ctx context.Context, doer Doer, target string, cfg effective) (string, error) {
	if shouldIgnoreLink(target, cfg.linkIgnore) {
		return "ok", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	if err != nil {
		return "fail", err
	}
	resp, err := doer.Do(req)
	if err != nil || (resp != nil && resp.StatusCode >= http.StatusBadRequest) {
		if resp != nil {
			resp.Body.Close()
		}
		req2, err2 := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err2 != nil {
			if cfg.strictExternal {
				return "fail", err2
			}
			return "warn", err2
		}
		resp, err = doer.Do(req2)
	}
	if err != nil {
		if cfg.strictExternal {
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

func shouldIgnoreLink(target string, ignore []string) bool {
	for _, pattern := range ignore {
		if pattern == "" {
			continue
		}
		if matchPattern(pattern, target) {
			return true
		}
	}
	return strings.Contains(target, "localhost") || strings.Contains(target, "127.0.0.1")
}
