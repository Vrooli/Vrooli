package runtimeprofile

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
)

// Handlers owns the `resource-ollama runtime ...` commands.
type Handlers struct {
	GetEnv func(string) string
	Stdout io.Writer
	Stderr io.Writer
	Now    func() time.Time
}

// Default returns Handlers wired to the process environment.
func Default() *Handlers {
	return &Handlers{GetEnv: os.Getenv, Stdout: os.Stdout, Stderr: os.Stderr, Now: time.Now}
}

// Commands returns the runtime subcommand group.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "runtime",
		Description: "Inspect the declared Ollama runner profile and detect runner reload drift",
		Subcommands: []cliapp.Command{
			{
				Name:        "profile",
				Description: "Print the declared runner options every caller receives",
				Usage:       "resource-ollama runtime profile [--json]",
				Run:         h.ProfileCommand,
			},
			{
				Name:        "status",
				Description: "Count runner starts, option-drift restarts, evictions, thread-budget misses and oversized embeddings in the daemon log",
				Usage:       "resource-ollama runtime status [--window 1h] [--log <service.log>] [--json] [--fail-on-drift]",
				Run:         h.Status,
			},
		},
	}
}

// ProfileCommand prints the embedded profile.
func (h *Handlers) ProfileCommand(args []string) error {
	fs := flag.NewFlagSet("runtime profile", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p := Current()
	if *asJSON {
		return cliout.NewCompactEncoder(h.Stdout).Encode(p)
	}
	env, _ := p.ServerEnvironment()
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintf(h.Stdout, "server: context_tokens=%d cpu_threads=%d\n", p.Server.ContextTokens, p.Server.CPUThreads)
	for _, name := range names {
		fmt.Fprintf(h.Stdout, "daemon env: %s=%s\n", name, env[name])
	}
	fmt.Fprintf(h.Stdout, "callers never send: %s\n", strings.Join(p.RequestLoadOptions, ", "))
	fmt.Fprintf(h.Stdout, "embedding overflow: %s\n", p.Embedding.Overflow)
	refs := make([]string, 0, len(p.Models))
	for ref := range p.Models {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		fmt.Fprintf(h.Stdout, "model %s: context_tokens=%d\n", ref, p.Models[ref].ContextTokens)
	}
	return nil
}

// RunnerReport summarizes one model's runner starts in the window.
type RunnerReport struct {
	Model               string         `json:"model"`
	Blob                string         `json:"blob"`
	Starts              int            `json:"starts"`
	OptionDriftRestarts int            `json:"option_drift_restarts"`
	Signatures          map[string]int `json:"signatures"`
	ThreadsObserved     map[string]int `json:"threads_observed,omitempty"`
}

// StatusReport is the drift signal for the declared runtime profile.
type StatusReport struct {
	Verdict              string         `json:"verdict"`
	WindowStart          time.Time      `json:"window_start"`
	WindowEnd            time.Time      `json:"window_end"`
	Logs                 []string       `json:"logs"`
	RunnerStarts         int            `json:"runner_starts"`
	OptionDriftRestarts  int            `json:"option_drift_restarts"`
	Evictions            int            `json:"evictions"`
	ThreadBudgetMisses   int            `json:"thread_budget_misses"`
	EmbedInputTooLarge   int            `json:"embed_input_too_large"`
	EdgeNormalizedCalls  int            `json:"edge_normalized_calls"`
	Runners              []RunnerReport `json:"runners"`
	DeclaredCPUThreads   int            `json:"declared_cpu_threads"`
	DeclaredContextLimit int            `json:"declared_context_tokens"`
}

// Status scans the daemon and edge logs for the configured window.
func (h *Handlers) Status(args []string) error {
	fs := flag.NewFlagSet("runtime status", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	window := fs.Duration("window", time.Hour, "How far back to scan")
	logPath := fs.String("log", "", "Ollama daemon log (default: the managed service.log and its rotation)")
	edgeLog := fs.String("edge-log", "", "Activity edge log (default: the managed activity-edge.log)")
	asJSON := fs.Bool("json", false, "Emit JSON")
	failOnDrift := fs.Bool("fail-on-drift", false, "Exit non-zero when the verdict is drift")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *window <= 0 {
		return fmt.Errorf("--window must be positive")
	}
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}
	logs := []string{*logPath}
	if strings.TrimSpace(*logPath) == "" {
		base := h.defaultServiceLog()
		logs = []string{base + ".1", base}
	}
	edge := strings.TrimSpace(*edgeLog)
	if edge == "" {
		edge = h.defaultEdgeLog()
	}
	report, err := ScanLogs(Current(), logs, edge, now.Add(-*window), now)
	if err != nil {
		return err
	}
	if *asJSON {
		if err := cliout.NewCompactEncoder(h.Stdout).Encode(report); err != nil {
			return err
		}
	} else {
		writeStatusText(h.Stdout, report)
	}
	if *failOnDrift && report.Verdict != "ok" {
		return fmt.Errorf("Ollama runtime drift detected: %d option-drift restarts, %d thread-budget misses, %d oversized embeddings", report.OptionDriftRestarts, report.ThreadBudgetMisses, report.EmbedInputTooLarge)
	}
	return nil
}

func writeStatusText(w io.Writer, r StatusReport) {
	fmt.Fprintf(w, "verdict: %s (%s .. %s)\n", r.Verdict, r.WindowStart.Format(time.RFC3339), r.WindowEnd.Format(time.RFC3339))
	fmt.Fprintf(w, "runner starts: %d  option-drift restarts: %d  evictions: %d  thread-budget misses: %d (declared %d)\n", r.RunnerStarts, r.OptionDriftRestarts, r.Evictions, r.ThreadBudgetMisses, r.DeclaredCPUThreads)
	fmt.Fprintf(w, "oversized embedding inputs: %d  caller load options replaced by the edge: %d\n", r.EmbedInputTooLarge, r.EdgeNormalizedCalls)
	for _, runner := range r.Runners {
		fmt.Fprintf(w, "  %s: starts=%d drift=%d", runner.Model, runner.Starts, runner.OptionDriftRestarts)
		sigs := make([]string, 0, len(runner.Signatures))
		for sig, count := range runner.Signatures {
			sigs = append(sigs, fmt.Sprintf("[%s]x%d", sig, count))
		}
		sort.Strings(sigs)
		fmt.Fprintf(w, " %s\n", strings.Join(sigs, " "))
	}
}

func (h *Handlers) getenv(key string) string {
	if h.GetEnv == nil {
		return os.Getenv(key)
	}
	return h.GetEnv(key)
}

func (h *Handlers) defaultServiceLog() string {
	state := strings.TrimSpace(h.getenv("XDG_STATE_HOME"))
	if state == "" {
		state = filepath.Join(h.getenv("HOME"), ".local", "state")
	}
	return filepath.Join(state, "logs", "vrooli", "resources", "ollama", "service.log")
}

func (h *Handlers) defaultEdgeLog() string {
	return filepath.Join(h.getenv("HOME"), ".vrooli", "processes", "resources", "ollama", "activity-edge.log")
}

var (
	logTimePattern   = regexp.MustCompile(`^time=(\S+)`)
	ginTimePattern   = regexp.MustCompile(`^\[GIN\] (\d{4}/\d{2}/\d{2} - \d{2}:\d{2}:\d{2})`)
	cmdPattern       = regexp.MustCompile(`msg="starting llama-server" cmd="([^"]*)"`)
	threadsPattern   = regexp.MustCompile(`^system_info: n_threads = (\d+)`)
	edgeLinePattern  = regexp.MustCompile(`ollama runtime profile: replaced caller load options .*at=(\S+).* requests=(\d+)`)
	signatureFlags   = []string{"-c", "-b", "-ub", "-t", "-tb", "-ngl", "-mg"}
	signatureSwitchs = []string{"--no-mmap"}
)

type runnerState struct {
	report        *RunnerReport
	lastSignature string
}

// ScanLogs computes the status report from daemon logs (oldest first) and the
// edge log. Missing files are skipped; an empty scan is an ok verdict.
func ScanLogs(p Profile, logs []string, edgeLog string, start, end time.Time) (StatusReport, error) {
	report := StatusReport{WindowStart: start, WindowEnd: end, DeclaredCPUThreads: p.Server.CPUThreads, DeclaredContextLimit: p.Server.ContextTokens}
	runners := map[string]*runnerState{}
	var order []string
	var lastStart *runnerState
	modelNames := map[string]string{}
	for _, path := range logs {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return StatusReport{}, err
		}
		report.Logs = append(report.Logs, path)
		var current time.Time
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if m := logTimePattern.FindStringSubmatch(line); m != nil {
				if t, err := time.Parse(time.RFC3339Nano, m[1]); err == nil {
					current = t
				}
			} else if m := ginTimePattern.FindStringSubmatch(line); m != nil {
				if t, err := time.ParseInLocation("2006/01/02 - 15:04:05", m[1], time.Local); err == nil {
					current = t
				}
			}
			if current.IsZero() || current.Before(start) || current.After(end) {
				continue
			}
			if m := cmdPattern.FindStringSubmatch(line); m != nil {
				blob, modelsDir, signature := parseRunnerCommand(m[1])
				state := runners[blob]
				if state == nil {
					name := blob
					if modelsDir != "" {
						if len(modelNames) == 0 {
							modelNames = manifestModelNames(modelsDir)
						}
						if resolved := modelNames[blob]; resolved != "" {
							name = resolved
						}
					}
					state = &runnerState{report: &RunnerReport{Model: name, Blob: shortBlob(blob), Signatures: map[string]int{}}}
					runners[blob] = state
					order = append(order, blob)
				}
				state.report.Starts++
				state.report.Signatures[signature]++
				report.RunnerStarts++
				if state.lastSignature != "" && state.lastSignature != signature {
					state.report.OptionDriftRestarts++
					report.OptionDriftRestarts++
				}
				state.lastSignature = signature
				lastStart = state
				continue
			}
			if m := threadsPattern.FindStringSubmatch(line); m != nil && lastStart != nil {
				if lastStart.report.ThreadsObserved == nil {
					lastStart.report.ThreadsObserved = map[string]int{}
				}
				lastStart.report.ThreadsObserved[m[1]]++
				if n, err := strconv.Atoi(m[1]); err == nil && n != p.Server.CPUThreads {
					report.ThreadBudgetMisses++
				}
				lastStart = nil
				continue
			}
			if strings.Contains(line, "evicting") {
				report.Evictions++
			}
			if strings.Contains(line, "is too large to process") {
				report.EmbedInputTooLarge++
			}
		}
		closeErr := file.Close()
		if err := scanner.Err(); err != nil {
			return StatusReport{}, err
		}
		if closeErr != nil {
			return StatusReport{}, closeErr
		}
	}
	report.EdgeNormalizedCalls = scanEdgeLog(edgeLog, start, end)
	for _, blob := range order {
		report.Runners = append(report.Runners, *runners[blob].report)
	}
	report.Verdict = "ok"
	if report.OptionDriftRestarts > 0 || report.ThreadBudgetMisses > 0 || report.EmbedInputTooLarge > 0 {
		report.Verdict = "drift"
	}
	return report, nil
}

func scanEdgeLog(path string, start, end time.Time) int {
	if strings.TrimSpace(path) == "" {
		return 0
	}
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	total := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		m := edgeLinePattern.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, m[1])
		if err != nil || t.Before(start) || t.After(end) {
			continue
		}
		n, _ := strconv.Atoi(m[2])
		total += n
	}
	return total
}

// parseRunnerCommand returns the model blob digest, the models directory and
// the load-option signature of one llama-server command line.
func parseRunnerCommand(cmd string) (blob, modelsDir, signature string) {
	fields := strings.Fields(cmd)
	values := map[string]string{}
	switches := map[string]bool{}
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if field == "--model" && i+1 < len(fields) {
			path := fields[i+1]
			blob = strings.TrimPrefix(filepath.Base(path), "sha256-")
			if filepath.Base(filepath.Dir(path)) == "blobs" {
				modelsDir = filepath.Dir(filepath.Dir(path))
			}
			i++
			continue
		}
		for _, flagName := range signatureFlags {
			if field == flagName && i+1 < len(fields) {
				values[flagName] = fields[i+1]
			}
		}
		for _, switchName := range signatureSwitchs {
			if field == switchName {
				switches[switchName] = true
			}
		}
	}
	parts := make([]string, 0, len(signatureFlags)+len(signatureSwitchs))
	for _, flagName := range signatureFlags {
		value := values[flagName]
		if value == "" {
			value = "default"
		}
		parts = append(parts, strings.TrimLeft(flagName, "-")+"="+value)
	}
	for _, switchName := range signatureSwitchs {
		parts = append(parts, strings.TrimLeft(switchName, "-")+"="+strconv.FormatBool(switches[switchName]))
	}
	return blob, modelsDir, strings.Join(parts, " ")
}

func shortBlob(blob string) string {
	if len(blob) > 12 {
		return blob[:12]
	}
	return blob
}

// manifestModelNames maps model weight blob digests to Ollama model names.
func manifestModelNames(modelsDir string) map[string]string {
	names := map[string]string{}
	root := filepath.Join(modelsDir, "manifests")
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var manifest struct {
			Layers []struct {
				MediaType string `json:"mediaType"`
				Digest    string `json:"digest"`
			} `json:"layers"`
		}
		if json.Unmarshal(raw, &manifest) != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 {
			return nil
		}
		name := strings.Join(parts[:len(parts)-1], "/") + ":" + parts[len(parts)-1]
		name = strings.TrimPrefix(name, "registry.ollama.ai/library/")
		for _, layer := range manifest.Layers {
			if layer.MediaType == "application/vnd.ollama.image.model" {
				names[strings.TrimPrefix(layer.Digest, "sha256:")] = name
			}
		}
		return nil
	})
	return names
}
