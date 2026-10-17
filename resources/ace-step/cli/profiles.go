package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
)

type modelProfile struct {
	ID      string `json:"id"`
	Variant string `json:"variant"`
	Planner string `json:"planner"`
}

var modelProfiles = []modelProfile{
	{"fast", "acestep-v15-turbo", "acestep-5Hz-lm-0.6B"},
	{"balanced", "acestep-v15-turbo", "acestep-5Hz-lm-1.7B"},
	{"detail", "acestep-v15-sft", "acestep-5Hz-lm-1.7B"},
}

type profileHandlers struct {
	GetEnv  func(string) string
	Out     io.Writer
	Err     io.Writer
	Info    func(context.Context) (map[string]any, error)
	Restart func(context.Context) error
}

func defaultProfileHandlers() *profileHandlers {
	return &profileHandlers{
		GetEnv: os.Getenv, Out: os.Stdout, Err: os.Stderr,
		Info: func(ctx context.Context) (map[string]any, error) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8895/health", nil)
			if err != nil {
				return nil, err
			}
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			data, err := io.ReadAll(io.LimitReader(response.Body, 16385))
			if err != nil {
				return nil, err
			}
			if len(data) > 16384 {
				return nil, errors.New("health observation exceeds 16 KiB")
			}
			var observed map[string]any
			if err := json.Unmarshal(data, &observed); err != nil {
				return nil, err
			}
			return observed, nil
		},
		Restart: func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "vrooli", "resource", "restart", appName)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

func profileCommands(h *profileHandlers) cliapp.SubcommandGroup {
	if h == nil {
		h = defaultProfileHandlers()
	}
	return cliapp.SubcommandGroup{Name: "profiles", Description: "Inspect and select fixed ACE model/planner profiles; selection is not a quality verdict", Subcommands: []cliapp.Command{
		{Name: "list", Usage: "resource-ace-step profiles list [--json]", Description: "List existing pinned model/planner combinations", Run: h.List},
		{Name: "status", Usage: "resource-ace-step profiles status [--json]", Description: "Compare desired selection with observed loaded identity", Run: h.Status},
		{Name: "activate", Usage: "resource-ace-step profiles activate --profile <fast|balanced|detail> [--no-restart] [--json]", Description: "Persist selection and normally restart ACE only", Run: h.Activate},
	}}
}

func profileByID(id string) (modelProfile, error) {
	for _, p := range modelProfiles {
		if p.ID == id {
			return p, nil
		}
	}
	return modelProfile{}, fmt.Errorf("unknown ACE profile %q; choose fast, balanced or detail", id)
}

func profileFor(variant, planner string) (modelProfile, bool) {
	for _, p := range modelProfiles {
		if p.Variant == variant && p.Planner == planner {
			return p, true
		}
	}
	return modelProfile{}, false
}

func (h *profileHandlers) statePath() string {
	if root := strings.TrimSpace(h.GetEnv("RESOURCE_DATA_DIR")); root != "" {
		return filepath.Join(root, "active.env")
	}
	root := strings.TrimSpace(h.GetEnv("XDG_DATA_HOME"))
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(root, "vrooli", "resources", appName, "active.env")
}

func (h *profileHandlers) desired() (modelProfile, error) {
	file, err := os.Open(h.statePath())
	if os.IsNotExist(err) {
		return modelProfiles[0], nil
	}
	if err != nil {
		return modelProfile{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return modelProfile{}, err
	}
	if len(data) > 4096 {
		return modelProfile{}, errors.New("ACE profile state exceeds 4 KiB")
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || (key != "ACE_STEP_VARIANT" && key != "ACE_STEP_PLANNER") {
			return modelProfile{}, errors.New("invalid ACE profile state")
		}
		if _, duplicate := values[key]; duplicate {
			return modelProfile{}, errors.New("duplicate ACE profile state field")
		}
		values[key] = value
	}
	p, ok := profileFor(values["ACE_STEP_VARIANT"], values["ACE_STEP_PLANNER"])
	if !ok {
		return modelProfile{}, errors.New("ACE profile state has unsupported or incomplete model/planner pair")
	}
	return p, nil
}

func loadedProfile(info map[string]any) (modelProfile, bool) {
	loaded, _ := info["model_loaded"].(bool)
	errorText, _ := info["error"].(string)
	if !loaded || errorText != "" {
		return modelProfile{}, false
	}
	variant, _ := info["loaded_variant"].(string)
	planner, _ := info["loaded_planner"].(string)
	return profileFor(variant, planner)
}

func (h *profileHandlers) flags(name string, args []string) (*flag.FlagSet, *bool, error) {
	fs := flag.NewFlagSet("profiles "+name, flag.ContinueOnError)
	fs.SetOutput(h.Err)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	if fs.NArg() != 0 {
		return nil, nil, errors.New("unexpected positional arguments")
	}
	return fs, jsonOut, nil
}

func (h *profileHandlers) output(value any, compact bool) error {
	if compact {
		return cliout.NewEncoder(h.Out).Encode(value)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.Out, string(data))
	return err
}

func (h *profileHandlers) List(args []string) error {
	_, compact, err := h.flags("list", args)
	if err != nil {
		return err
	}
	return h.output(map[string]any{"profiles": modelProfiles, "default_profile": "fast", "note": "All profiles use existing pinned inputs; detail is an experiment, not a quality guarantee."}, *compact)
}

func (h *profileHandlers) Status(args []string) error {
	_, compact, err := h.flags("status", args)
	if err != nil {
		return err
	}
	desired, err := h.desired()
	if err != nil {
		return err
	}
	result := map[string]any{"desired": desired, "state_path": h.statePath(), "loaded": nil, "restart_required": nil}
	info, err := h.Info(context.Background())
	if err != nil {
		result["observation_error"] = err.Error()
		return h.output(result, *compact)
	}
	if actual, known := loadedProfile(info); known {
		result["loaded"] = actual
		result["restart_required"] = actual != desired
	} else {
		result["observation_error"] = "loaded model/planner identity unavailable or not ready"
	}
	return h.output(result, *compact)
}

func (h *profileHandlers) Activate(args []string) error {
	fs := flag.NewFlagSet("profiles activate", flag.ContinueOnError)
	fs.SetOutput(h.Err)
	id := fs.String("profile", "", "Fixed profile ID")
	noRestart := fs.Bool("no-restart", false, "Persist desired selection only")
	compact := fs.Bool("json", false, "Emit compact JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	desired, err := profileByID(*id)
	if err != nil {
		return err
	}
	data := []byte("ACE_STEP_VARIANT=" + desired.Variant + "\nACE_STEP_PLANNER=" + desired.Planner + "\n")
	if err := config.WriteOwnedFileAtomic(h.statePath(), data, 0o600); err != nil {
		return err
	}
	if !*noRestart {
		if h.Restart == nil {
			return errors.New("ACE restart owner is unavailable; desired selection persisted")
		}
		if err := h.Restart(context.Background()); err != nil {
			return fmt.Errorf("desired profile persisted; managed ACE restart failed: %w", err)
		}
		info, err := h.Info(context.Background())
		if err != nil {
			return fmt.Errorf("desired profile persisted; loaded identity unverified: %w", err)
		}
		actual, known := loadedProfile(info)
		if !known || actual != desired {
			return errors.New("desired profile persisted; loaded identity did not acknowledge selected profile")
		}
	}
	return h.output(map[string]any{"desired": desired, "state_path": h.statePath(), "restarted": !*noRestart, "activation": map[bool]string{true: "desired-only", false: "loaded-verified"}[*noRestart]}, *compact)
}
