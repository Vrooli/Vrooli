// Package localmodel prepares an Ollama policy role for OpenCode by composing
// the Ollama and OpenCode resource owners' supported operations.
package localmodel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/configcli"
)

const defaultTimeout = 45 * time.Minute
const reservationTTLGrace = 15 * time.Minute
const maxTimeout = 24 * time.Hour

type CommandRunner interface {
	Run(context.Context, string, []string, io.Writer) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args []string, progress io.Writer) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	if progress != nil {
		cmd.Stdout = io.MultiWriter(&stdout, progress)
	} else {
		cmd.Stdout = &stdout
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return stdout.Bytes(), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, detail)
		}
		return stdout.Bytes(), fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}

type Hooks struct {
	Runner          CommandRunner
	CommitSelection func(context.Context, string) error
	Progress        io.Writer
}

type Receipt struct {
	Status        string `json:"status"`
	Role          string `json:"role"`
	Model         string `json:"model"`
	SelectedModel string `json:"selected_model"`
	Stage         string `json:"stage,omitempty"`
	Error         string `json:"error,omitempty"`
}

type roleResolution struct {
	Model string `json:"model"`
}

type reservationReceipt struct {
	ID             string `json:"id"`
	ClaimID        string `json:"claim_id"`
	Role           string `json:"role"`
	Model          string `json:"model"`
	RequestedBytes int64  `json:"requested_bytes"`
	CapacityBytes  int64  `json:"capacity_bytes"`
	AdmissionBytes int64  `json:"admission_bytes"`
}

type doctorResult struct {
	Pass   bool `json:"pass"`
	Models []struct {
		Role             string   `json:"role"`
		Model            string   `json:"model"`
		Pass             bool     `json:"pass"`
		Installed        bool     `json:"installed"`
		LiveCapabilities []string `json:"live_capabilities"`
	} `json:"models"`
}

type chatResult struct {
	Response  string `json:"response"`
	EvalCount int    `json:"eval_count"`
}

type toolProbeResult struct {
	Model         string `json:"model"`
	SupportsTools bool   `json:"supports_tools"`
}

// Prepare resolves a policy role, obtains a model-sized Ollama-owned capacity
// reservation through the shared companion, acquires the model through Ollama's
// disk-admitted ensure path, checks chat and tool readiness, and selects it.
func Prepare(ctx context.Context, role string, hooks Hooks) (receipt Receipt, returnErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	role = strings.TrimSpace(role)
	receipt = Receipt{Status: "preparing", Role: role}
	if role == "" {
		return failed(receipt, "validate", errors.New("--role is required"))
	}
	if hooks.CommitSelection == nil {
		return failed(receipt, "select", errors.New("OpenCode local model projection and selection are unavailable"))
	}
	if hooks.Runner == nil {
		hooks.Runner = execRunner{}
	}
	if hooks.Progress == nil {
		hooks.Progress = io.Discard
	}
	run := func(stage, name string, args ...string) ([]byte, error) {
		receipt.Stage = stage
		fmt.Fprintf(hooks.Progress, "[%s] running %s\n", stage, name)
		return hooks.Runner.Run(ctx, name, args, nil)
	}
	runWithProgress := func(stage, name string, args ...string) ([]byte, error) {
		receipt.Stage = stage
		fmt.Fprintf(hooks.Progress, "[%s] running %s\n", stage, name)
		return hooks.Runner.Run(ctx, name, args, hooks.Progress)
	}
	resolved, err := hooks.Runner.Run(ctx, "resource-ollama", []string{"policy", "resolve", "--role", role, "--json"}, nil)
	if err != nil {
		return failed(receipt, "resolve", err)
	}
	var roleInfo roleResolution
	if err := json.Unmarshal(resolved, &roleInfo); err != nil {
		return failed(receipt, "resolve", fmt.Errorf("decode Ollama role resolution: %w", err))
	}
	receipt.Model = strings.TrimSpace(roleInfo.Model)
	if receipt.Model == "" {
		return failed(receipt, "resolve", errors.New("Ollama policy returned an empty model"))
	}

	reservationBytes, err := run("capacity-reserve", "resource-ollama", "capacity", "reserve-model", "--role", role, "--ttl-seconds", strconv.Itoa(reservationTTLSeconds(ctx)), "--json")
	if err != nil {
		return failed(receipt, "capacity-reserve", err)
	}
	var reservation reservationReceipt
	if err := json.Unmarshal(reservationBytes, &reservation); err != nil {
		return failed(receipt, "capacity-reserve", fmt.Errorf("decode Ollama model capacity reservation: %w", err))
	}
	reservationSettled := false
	defer func() {
		if reservationSettled {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		_, cleanupErr := hooks.Runner.Run(cleanupCtx, "resource-ollama", []string{"capacity", "release-model-reservation", "--id", reservation.ID, "--json"}, nil)
		if cleanupErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("release Ollama model capacity reservation %s: %w", reservation.ID, cleanupErr))
			receipt.Status = "failed"
			receipt.Error = returnErr.Error()
		}
	}()
	if reservation.ID == "" || reservation.Role != role || reservation.Model != receipt.Model || reservation.RequestedBytes <= 0 {
		return failed(receipt, "capacity-reserve", errors.New("Ollama returned an incomplete or mismatched model capacity reservation"))
	}

	confirmedBytes, err := run("capacity-admission", "resource-ollama", "capacity", "confirm-model-reservation", "--id", reservation.ID, "--wait-seconds", "30", "--json")
	if err != nil {
		return failed(receipt, "capacity-admission", fmt.Errorf("Ollama model-sized capacity admission failed: %w", err))
	}
	var confirmed reservationReceipt
	if err := json.Unmarshal(confirmedBytes, &confirmed); err != nil {
		return failed(receipt, "capacity-admission", fmt.Errorf("decode admitted Ollama model capacity: %w", err))
	}
	if confirmed.ID != reservation.ID || confirmed.ClaimID == "" || confirmed.Role != reservation.Role || confirmed.Model != reservation.Model || confirmed.RequestedBytes != reservation.RequestedBytes || confirmed.CapacityBytes < reservation.RequestedBytes || confirmed.AdmissionBytes < 0 || confirmed.AdmissionBytes > confirmed.CapacityBytes {
		return failed(receipt, "capacity-admission", errors.New("Ollama returned an incomplete model-sized capacity admission"))
	}
	if _, err := run("runtime-start", "vrooli", "resource", "start", "ollama", "--no-reclaim", "--capacity-claim-id", confirmed.ClaimID, "--capacity-claim-min-bytes", strconv.FormatInt(confirmed.CapacityBytes, 10)); err != nil {
		return failed(receipt, "runtime-start", fmt.Errorf("start Ollama under its admitted model-sized capacity claim: %w", err))
	}
	ensureConfig, _ := json.Marshal(map[string]any{"model_roles": []map[string]string{{"role": role}}})
	ensureTimeout := defaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		ensureTimeout = time.Until(deadline).Truncate(time.Second)
		if ensureTimeout < time.Second {
			return failed(receipt, "acquire", context.DeadlineExceeded)
		}
	}
	if _, err := runWithProgress("acquire", "resource-ollama", "ensure", "--config-base64", base64.StdEncoding.EncodeToString(ensureConfig), "--timeout-seconds", strconv.Itoa(int(ensureTimeout.Seconds()))); err != nil {
		return failed(receipt, "acquire", err)
	}

	doctorBytes, err := run("model-check", "resource-ollama", "models", "doctor", "--role", role, "--json")
	if err != nil {
		return failed(receipt, "model-check", err)
	}
	var doctor doctorResult
	if err := json.Unmarshal(doctorBytes, &doctor); err != nil {
		return failed(receipt, "model-check", fmt.Errorf("decode Ollama model doctor: %w", err))
	}
	if !doctor.Pass || len(doctor.Models) != 1 {
		return failed(receipt, "model-check", errors.New("Ollama model doctor did not return one passing model"))
	}
	checked := doctor.Models[0]
	if checked.Role != role || checked.Model != receipt.Model || !checked.Installed || !checked.Pass || len(checked.LiveCapabilities) == 0 {
		return failed(receipt, "model-check", fmt.Errorf("Ollama model doctor did not prove installed live capabilities for %q", receipt.Model))
	}

	toolBytes, err := run("tool-check", "resource-ollama", "models", "probe-tools", "--role", role, "--json")
	if err != nil {
		return failed(receipt, "tool-check", err)
	}
	var tools toolProbeResult
	if err := json.Unmarshal(toolBytes, &tools); err != nil {
		return failed(receipt, "tool-check", fmt.Errorf("decode Ollama tool probe: %w", err))
	}
	if tools.Model != receipt.Model || !tools.SupportsTools {
		return failed(receipt, "tool-check", fmt.Errorf("Ollama model %q did not prove structured tool calling", receipt.Model))
	}

	chatBytes, err := run("chat-smoke", "resource-ollama", "gateway", "chat", "--role", role, "--prompt", "Reply with a short greeting.", "--max-tokens", "16", "--json")
	if err != nil {
		return failed(receipt, "chat-smoke", err)
	}
	var chat chatResult
	if err := json.Unmarshal(chatBytes, &chat); err != nil {
		return failed(receipt, "chat-smoke", fmt.Errorf("decode Ollama chat smoke: %w", err))
	}
	if strings.TrimSpace(chat.Response) == "" || chat.EvalCount == 0 {
		return failed(receipt, "chat-smoke", errors.New("Ollama chat smoke returned no generated text"))
	}
	releaseBytes, err := run("capacity-handoff", "resource-ollama", "capacity", "release-model-reservation", "--id", reservation.ID, "--json")
	if err != nil {
		return failed(receipt, "capacity-handoff", err)
	}
	var released struct {
		Released bool `json:"released"`
	}
	if err := json.Unmarshal(releaseBytes, &released); err != nil || !released.Released {
		if err == nil {
			err = errors.New("Ollama did not confirm model reservation handoff")
		}
		return failed(receipt, "capacity-handoff", err)
	}
	reservationSettled = true

	receipt.Stage = "select"
	fmt.Fprintln(hooks.Progress, "[select] projecting and selecting the prepared model in OpenCode")
	if err := hooks.CommitSelection(ctx, receipt.Model); err != nil {
		return failed(receipt, "select", err)
	}
	receipt.Status = "ready"
	receipt.Stage = "ready"
	receipt.SelectedModel = "ollama/" + receipt.Model
	return receipt, nil
}

func failed(receipt Receipt, stage string, err error) (Receipt, error) {
	receipt.Status = "failed"
	receipt.Stage = stage
	receipt.Error = err.Error()
	return receipt, err
}

type Handlers struct {
	Hooks  Hooks
	Stdout io.Writer
	Stderr io.Writer
}

func Default() *Handlers {
	return &Handlers{Hooks: Hooks{Runner: execRunner{}}, Stdout: os.Stdout, Stderr: os.Stderr}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "local-model",
		Description: "Prepare and select local Ollama models for OpenCode",
		Subcommands: []cliapp.Command{{
			Name:        "select",
			Description: "Admit capacity, install, verify, and select a local Ollama role",
			Usage:       "resource-opencode local-model select --role <ollama-policy-role> [--timeout-seconds <seconds>] [--json]",
			Run:         h.Select,
		}},
	}
}

func (h *Handlers) Select(args []string) error {
	fs := flag.NewFlagSet("local-model select", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Ollama policy role to prepare")
	timeoutSeconds := fs.Int("timeout-seconds", int(defaultTimeout.Seconds()), "Total operation timeout")
	asJSON := fs.Bool("json", false, "Emit a machine-readable final receipt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *timeoutSeconds <= 0 {
		return errors.New("--timeout-seconds must be positive")
	}
	if *timeoutSeconds > int(maxTimeout.Seconds()) {
		return fmt.Errorf("--timeout-seconds must not exceed %d", int(maxTimeout.Seconds()))
	}
	if h.Hooks.Runner == nil {
		h.Hooks.Runner = execRunner{}
	}
	progress := h.Stdout
	if *asJSON {
		progress = h.Stderr
	}
	h.Hooks.Progress = progress
	if h.Hooks.CommitSelection == nil {
		configOut := h.Stdout
		if *asJSON {
			configOut = io.Discard
		}
		h.Hooks.CommitSelection = func(ctx context.Context, model string) error {
			config := configcli.Default()
			config.Stdout, config.Stderr = configOut, h.Stderr
			return config.PrepareLocalModelContext(ctx, model)
		}
	}
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(signalCtx, time.Duration(*timeoutSeconds)*time.Second)
	defer cancel()
	receipt, err := Prepare(ctx, *role, h.Hooks)
	if *asJSON {
		if encodeErr := json.NewEncoder(h.Stdout).Encode(receipt); encodeErr != nil {
			return encodeErr
		}
	} else if err == nil {
		fmt.Fprintf(h.Stdout, "Ready: %s is selected in OpenCode\n", receipt.SelectedModel)
	}
	return err
}

func deadline(ctx context.Context) time.Time {
	deadline, ok := ctx.Deadline()
	if !ok {
		return time.Now().Add(defaultTimeout)
	}
	return deadline
}

func reservationTTLSeconds(ctx context.Context) int {
	return int(math.Ceil((time.Until(deadline(ctx)) + reservationTTLGrace).Seconds()))
}
