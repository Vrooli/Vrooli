// Package upstreamcheck implements the read-only `upstream-check` verb that
// every Vrooli coding-agent resource (opencode, codex, claude-code) shares.
//
// It compares the locally-installed upstream CLI version against the latest
// published release and reports one of: up-to-date | behind | ahead |
// unknown. It is intentionally agent-safe: a network or parse failure
// degrades to "unknown" and exit 0 — it never hard-fails the resource.
//
// This is the single shared home for the check. It lives in cli-core so the
// three agent resource CLIs (which already depend on cli-core/cliapp) import
// it directly rather than each carrying a verbatim copy.
package upstreamcheck

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SourceKind identifies how the latest upstream version is discovered.
type SourceKind string

const (
	// SourceGitHub resolves the latest GitHub release tag for "owner/repo".
	SourceGitHub SourceKind = "github"
	// SourceNPM resolves the latest published version for an npm package.
	SourceNPM SourceKind = "npm"
)

// Status is the relative-state verdict.
type Status string

const (
	StatusUpToDate Status = "up-to-date"
	StatusBehind   Status = "behind"
	StatusAhead    Status = "ahead"
	StatusUnknown  Status = "unknown"
)

// Config is the per-resource binding for the check.
type Config struct {
	// DisplayName labels the report (e.g. "opencode").
	DisplayName string
	// InstalledCmd runs the installed CLI's version probe (e.g.
	// ["opencode","--version"]).
	InstalledCmd []string
	// PinnedVersion mirrors resource.json upstream_cli.version_pinned.
	PinnedVersion string
	// SourceKind selects the discovery transport.
	SourceKind SourceKind
	// SourceID is "owner/repo" for GitHub or the package name for npm.
	SourceID string
}

// Handlers carries runtime dependencies; tests inject the seams.
type Handlers struct {
	Cfg Config
	// InstalledRunner returns the raw `--version` output. Defaults to exec.
	InstalledRunner func(ctx context.Context, args []string) (string, error)
	// LatestFetcher returns the latest upstream version string. Defaults to
	// an HTTP lookup against GitHub / the npm registry.
	LatestFetcher func(ctx context.Context, kind SourceKind, id string) (string, error)
	HTTPClient    *http.Client
	Timeout       time.Duration
	Stdout        io.Writer
	Stderr        io.Writer
}

// Default wires Handlers with the live exec + HTTP seams.
func Default(cfg Config) *Handlers {
	h := &Handlers{
		Cfg:        cfg,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Timeout:    10 * time.Second,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	h.InstalledRunner = defaultInstalledRunner
	h.LatestFetcher = h.fetchLatest
	return h
}

func defaultInstalledRunner(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty version command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Report is the structured result emitted by --json.
type Report struct {
	Name      string `json:"name"`
	Installed string `json:"installed"`
	Pinned    string `json:"pinned"`
	Latest    string `json:"latest"`
	Status    Status `json:"status"`
	Source    struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"source"`
	Note string `json:"note,omitempty"`
}

// Check is the verb entrypoint. It always exits 0 (read-only/agent-safe);
// only a flag-parse error is returned.
func (h *Handlers) Check(args []string) error {
	fs := flag.NewFlagSet("upstream-check", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit the result as JSON")
	pinned := fs.String("pinned-version", h.Cfg.PinnedVersion, "Override the pinned version for this check")
	if err := fs.Parse(args); err != nil {
		return err
	}

	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rep := Report{Name: h.Cfg.DisplayName, Pinned: strings.TrimSpace(*pinned), Status: StatusUnknown}
	rep.Source.Kind = string(h.Cfg.SourceKind)
	rep.Source.ID = h.Cfg.SourceID

	var notes []string

	if raw, err := h.InstalledRunner(ctx, h.Cfg.InstalledCmd); err != nil {
		notes = append(notes, fmt.Sprintf("installed version unavailable: %v", err))
	} else {
		rep.Installed = ExtractSemver(raw)
		if rep.Installed == "" {
			notes = append(notes, fmt.Sprintf("could not parse installed version from %q", raw))
		}
	}

	if h.LatestFetcher != nil {
		if latest, err := h.LatestFetcher(ctx, h.Cfg.SourceKind, h.Cfg.SourceID); err != nil {
			notes = append(notes, fmt.Sprintf("upstream lookup failed: %v", err))
		} else {
			rep.Latest = ExtractSemver(latest)
			if rep.Latest == "" {
				notes = append(notes, fmt.Sprintf("could not parse upstream version from %q", latest))
			}
		}
	}

	rep.Status = Compare(rep.Installed, rep.Latest)
	rep.Note = strings.Join(notes, "; ")

	if *asJSON {
		enc := json.NewEncoder(h.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	h.renderText(rep)
	return nil
}

func (h *Handlers) renderText(rep Report) {
	fmt.Fprintf(h.Stdout, "%s upstream check\n", rep.Name)
	fmt.Fprintf(h.Stdout, "  installed: %s\n", orDash(rep.Installed))
	fmt.Fprintf(h.Stdout, "  pinned:    %s\n", orDash(rep.Pinned))
	fmt.Fprintf(h.Stdout, "  latest:    %s (%s %s)\n", orDash(rep.Latest), rep.Source.Kind, rep.Source.ID)
	fmt.Fprintf(h.Stdout, "  status:    %s\n", rep.Status)
	if rep.Pinned != "" && rep.Latest != "" && rep.Pinned != rep.Latest {
		fmt.Fprintf(h.Stdout, "  note:      pin %s differs from latest %s — consider bumping resource.json\n", rep.Pinned, rep.Latest)
	}
	if rep.Note != "" {
		fmt.Fprintf(h.Stderr, "  warn:      %s\n", rep.Note)
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(unknown)"
	}
	return s
}

// fetchLatest is the live HTTP discovery for GitHub releases / npm registry.
func (h *Handlers) fetchLatest(ctx context.Context, kind SourceKind, id string) (string, error) {
	client := h.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	switch kind {
	case SourceGitHub:
		url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", id)
		var payload struct {
			TagName string `json:"tag_name"`
		}
		if err := getJSON(ctx, client, url, &payload); err != nil {
			return "", err
		}
		return payload.TagName, nil
	case SourceNPM:
		url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", id)
		var payload struct {
			Version string `json:"version"`
		}
		if err := getJSON(ctx, client, url, &payload); err != nil {
			return "", err
		}
		return payload.Version, nil
	default:
		return "", fmt.Errorf("unknown source kind %q", kind)
	}
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "vrooli-upstream-check")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

var semverRe = regexp.MustCompile(`\d+\.\d+\.\d+`)

// ExtractSemver pulls the first MAJOR.MINOR.PATCH token out of a raw
// version string (e.g. "codex-cli 0.131.0" → "0.131.0").
func ExtractSemver(raw string) string {
	return semverRe.FindString(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "v")))
}

// Compare returns the relative state of installed vs latest. An empty input
// yields StatusUnknown.
func Compare(installed, latest string) Status {
	if strings.TrimSpace(installed) == "" || strings.TrimSpace(latest) == "" {
		return StatusUnknown
	}
	switch cmpSemver(installed, latest) {
	case 0:
		return StatusUpToDate
	case -1:
		return StatusBehind
	case 1:
		return StatusAhead
	default:
		return StatusUnknown
	}
}

// cmpSemver compares two dotted numeric versions. Returns -1 if a<b, 0 if
// equal, 1 if a>b. Pre-release/build suffixes are ignored.
func cmpSemver(a, b string) int {
	pa := splitVersion(a)
	pb := splitVersion(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// Drop any pre-release/build metadata.
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}
