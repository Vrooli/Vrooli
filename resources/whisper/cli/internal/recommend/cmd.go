package recommend

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// envBudgetVar is the operator-visible knob. Read here, not in lib code,
// so the CLI is the single source of truth for budget resolution.
const envBudgetVar = "WHISPER_RESOURCE_BUDGET_PCT"

// Handlers owns the runtime dependencies for the recommend-model
// subcommand. Tests inject a FakeProbe; production wires SystemProbe.
type Handlers struct {
	Probe  HostProbe
	Stdout io.Writer
	Stderr io.Writer
	GetEnv func(string) string
	// Now is used for the JSON timestamp; tests pin it.
	Now func() time.Time
}

type HostProbe interface {
	Collect(ctx context.Context) (hostinventory.Snapshot, error)
}

// Default returns Handlers wired to the real OS probe.
func Default() *Handlers {
	probe := hostinventory.SystemCollector()
	return &Handlers{
		Probe:  probe,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		GetEnv: os.Getenv,
		Now:    time.Now,
	}
}

// Commands returns the `recommend-model` command for registration. The
// recommender lives at the top level so operators and the managed-service
// lifecycle can invoke it as a normal Whisper subcommand.
func Commands(h *Handlers) cliapp.Command {
	if h == nil {
		h = Default()
	}
	return cliapp.Command{
		Name:        "recommend-model",
		Description: "Recommend a Whisper model size for the current host",
		Usage:       "whisper recommend-model [--budget-pct N] [--json] [--explain]",
		Run:         h.Run,
	}
}

// jsonResult is the frozen schema documented in the plan. Fields here
// are the only public machine-readable contract.
type jsonResult struct {
	Model     string `json:"model"`
	Reason    string `json:"reason"`
	Host      host   `json:"host"`
	BudgetPct int    `json:"budget_pct"`
}

type host struct {
	OS       string  `json:"os"`
	Arch     string  `json:"arch"`
	CPUCores int     `json:"cpu_cores"`
	RAMGB    float64 `json:"ram_gb"`
	GPUs     []gpu   `json:"gpus"`
}

type gpu struct {
	Name   string  `json:"name"`
	VRAMGB float64 `json:"vram_gb"`
}

// Run implements cliapp.Command.Run. Defaults to human output; --json
// is opt-in and the only machine contract.
func (h *Handlers) Run(args []string) error {
	fs := flag.NewFlagSet("recommend-model", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	envOut := fs.Bool("env", false, "emit KEY=VALUE lines for shell tooling")
	explain := fs.Bool("explain", false, "include reasoning lines in human output")
	budgetFlag := fs.Int("budget-pct", 0, "percent of detected RAM/VRAM the recommender may spend")
	if err := fs.Parse(args); err != nil {
		return err
	}

	budgetPct := *budgetFlag
	if budgetPct == 0 {
		budgetPct = resolveBudgetPct(h.GetEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	caps, err := h.Probe.Collect(ctx)
	if err != nil {
		return fmt.Errorf("host inventory failed: %w", err)
	}

	// Honor a capacity-broker/operator model pin first (the degrade actuation
	// persists it): the pinned model overrides the host-derived recommendation so
	// the next managed-service start comes up at the smaller size. The pin is honored
	// before Pick so it works even on a host where Pick would otherwise error.
	var model Model
	var reason string
	if pinned, ok := ReadPin(h.GetEnv); ok {
		model = pinned
		reason = "model pinned by capacity broker/operator (overrides host recommendation)"
	} else {
		model, reason, err = Pick(caps, budgetPct)
		if err != nil {
			return err
		}
	}

	if *jsonOut {
		return writeJSON(h.Stdout, model, reason, caps, budgetPct)
	}
	if *envOut {
		return writeEnv(h.Stdout, model)
	}
	return writeHuman(h.Stdout, model, reason, caps, *explain)
}

func resolveBudgetPct(getEnv func(string) string) int {
	if getEnv == nil {
		return DefaultBudgetPct
	}
	v := getEnv(envBudgetVar)
	if v == "" {
		return DefaultBudgetPct
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 || n > 100 {
		return DefaultBudgetPct
	}
	return n
}

func writeJSON(w io.Writer, model Model, reason string, caps hostinventory.Snapshot, budgetPct int) error {
	r := jsonResult{
		Model:     string(model),
		Reason:    reason,
		BudgetPct: budgetPct,
		Host: host{
			OS:       caps.OS,
			Arch:     caps.Arch,
			CPUCores: caps.CPU.Cores,
			RAMGB:    float64(caps.Memory.TotalBytes) / float64(1<<30),
		},
	}
	for _, g := range caps.GPUs {
		r.Host.GPUs = append(r.Host.GPUs, gpu{
			Name:   g.Name,
			VRAMGB: float64(g.VRAMBytes) / float64(1<<30),
		})
	}
	return cliout.NewEncoder(w).Encode(r)
}

// writeEnv emits the KEY=VALUE lines consumed by operator tooling and the
// audio-tools LocalProvider model-truth seam.
func writeEnv(w io.Writer, model Model) error {
	_, err := fmt.Fprintf(w, "WHISPER_MODEL_SIZE=%s\nAUDIO_WHISPER_MODEL=%s\n", model, model)
	return err
}

func writeHuman(w io.Writer, model Model, reason string, caps hostinventory.Snapshot, explain bool) error {
	if _, err := fmt.Fprintf(w, "%s\n", model); err != nil {
		return err
	}
	if !explain {
		return nil
	}
	if _, err := fmt.Fprintf(w, "  reason: %s\n", reason); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "  host:   os=%s arch=%s cpu_cores=%d ram=%.1f GB gpus=%d\n",
		caps.OS, caps.Arch, caps.CPU.Cores,
		float64(caps.Memory.TotalBytes)/float64(1<<30), len(caps.GPUs))
	return err
}
