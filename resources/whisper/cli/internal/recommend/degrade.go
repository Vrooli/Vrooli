package recommend

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// DegradeHandlers owns the dependencies of the `capacity-degrade` verb (the
// adopter side of the capacity broker's degrade actuation, §8.9). Tests inject
// the seams; production wires the real shell + pin file.
type DegradeHandlers struct {
	Stdout io.Writer
	Stderr io.Writer
	GetEnv func(string) string
	// Exec runs a command and returns its combined output. Tests inject a fake so
	// the verb never shells out (no `vrooli`/docker in unit runs).
	Exec func(ctx context.Context, name string, args ...string) (string, error)
	// WritePinFn / ClearPinFn default to the package pin writers; tests may
	// override. The pin file path itself is env-overridable (EnvPinFile).
	WritePinFn func(getEnv func(string) string, model Model) error
	ClearPinFn func(getEnv func(string) string) error
}

// DefaultDegrade returns DegradeHandlers wired to the real shell + pin file.
func DefaultDegrade() *DegradeHandlers {
	return &DegradeHandlers{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		GetEnv:     os.Getenv,
		Exec:       realExec,
		WritePinFn: WritePin,
		ClearPinFn: ClearPin,
	}
}

func realExec(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// DegradeCommand returns the `capacity-degrade` command for registration. The
// broker invokes it as `whisper capacity-degrade --to <model>` when it needs to
// reclaim VRAM from an idle whisper; `--upshift --to <model>` recreates larger
// when headroom returns.
func DegradeCommand(h *DegradeHandlers) cliapp.Command {
	if h == nil {
		h = DefaultDegrade()
	}
	return cliapp.Command{
		Name:        "capacity-degrade",
		Description: "Resize Whisper to a smaller (or, with --upshift, larger) model at the capacity broker's request",
		Usage:       "whisper capacity-degrade --to <tiny|base|small|medium|large-v3> [--upshift] [--json]",
		Run:         h.Run,
	}
}

type degradeResult struct {
	Model    string `json:"model"`
	Upshift  bool   `json:"upshift"`
	Pinned   bool   `json:"pinned"`
	Recreate bool   `json:"recreated"`
	Message  string `json:"message"`
}

// Run implements the capacity-degrade verb.
func (h *DegradeHandlers) Run(args []string) error {
	fs := flag.NewFlagSet("capacity-degrade", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	to := fs.String("to", "", "target Whisper model label (profile step)")
	upshift := fs.Bool("upshift", false, "recreate at a LARGER model when idle headroom returns (clears the degrade pin)")
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	model, ok := ValidModel(*to)
	if !ok {
		return fmt.Errorf("--to must be a valid Whisper model (tiny|base|small|medium|large-v3), got %q", *to)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Refuse to recreate the container while transcription is active — recreating
	// would drop an in-flight request. The broker's planner already excludes
	// active/protected claims; this is the adopter-side defense in depth (§8.9).
	if active, owner := h.transcriptionActive(ctx); active {
		return fmt.Errorf("refusing to resize whisper: claim %q reports activity_state=active (transcription in progress)", owner)
	}

	// Persist the pin so the recreate (which harvests `recommend-model --env`)
	// comes up at the chosen model. Upshift clears the pin so the recommender
	// resumes host-derived sizing at the (larger) target ceiling.
	writePin := h.WritePinFn
	if writePin == nil {
		writePin = WritePin
	}
	clearPin := h.ClearPinFn
	if clearPin == nil {
		clearPin = ClearPin
	}
	if *upshift {
		if err := clearPin(h.GetEnv); err != nil {
			return fmt.Errorf("clear model pin: %w", err)
		}
	} else if err := writePin(h.GetEnv, model); err != nil {
		return fmt.Errorf("write model pin: %w", err)
	}

	// Recreate the container through the lifecycle so it re-harvests the model env
	// and reclaims/reallocates VRAM at the new size.
	recreated := true
	if out, err := h.Exec(ctx, "vrooli", "resource", "restart", "whisper"); err != nil {
		// The pin is persisted; a failed recreate is reported but not fatal — the
		// next lifecycle start will still come up at the pinned size.
		recreated = false
		fmt.Fprintf(h.Stderr, "warning: whisper recreate failed (pin persisted, applies on next start): %v\n%s\n", err, strings.TrimSpace(out))
	}

	res := degradeResult{
		Model:    string(model),
		Upshift:  *upshift,
		Pinned:   !*upshift,
		Recreate: recreated,
		Message:  fmt.Sprintf("whisper resized to %s", model),
	}
	if *upshift {
		res.Message = fmt.Sprintf("whisper upshift to %s (pin cleared)", model)
	}
	if *jsonOut {
		enc := json.NewEncoder(h.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	_, err := fmt.Fprintln(h.Stdout, res.Message)
	return err
}

// transcriptionActive queries the capacity ledger for whisper's active claim and
// reports whether any is currently transcribing. Best-effort: a query/parse
// failure returns false (the planner already gates on activity, so this never
// blocks the broker on a flaky read).
func (h *DegradeHandlers) transcriptionActive(ctx context.Context) (bool, string) {
	out, err := h.Exec(ctx, "vrooli", "capacity", "list", "--owner", "whisper", "--active", "--json")
	if err != nil {
		return false, ""
	}
	var payload struct {
		Claims []struct {
			OwnerID       string `json:"owner_id"`
			ActivityState string `json:"activity_state"`
		} `json:"claims"`
	}
	if jsonErr := json.Unmarshal([]byte(out), &payload); jsonErr != nil {
		return false, ""
	}
	for _, c := range payload.Claims {
		if c.ActivityState == "active" {
			return true, c.OwnerID
		}
	}
	return false, ""
}
