package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// userVerbs are the one-call verbs an operator or agent reaches for first:
// screenshot, record, open an app, fetch evidence, confirm a same-device
// relation. Each owns its preconditions (wake, evidence download) so a cold
// caller needs no second command.
func userVerbs(core *cliapp.ScenarioApp) []cliapp.Command {
	return []cliapp.Command{
		command("screenshot", "Take a screenshot of a TV, Google TV, Android phone or tablet in one call: wakes the screen, captures, saves the PNG and prints its path", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id or unique name, for example SmartTV 4K"}}, Flags: []cliapp.Flag{{Name: "home", Bool: true, Description: "press Home first so the capture shows the home screen; may change the foreground app"}, {Name: "no-wake", Bool: true, Description: "do not wake the screen first"}, {Name: "out", Description: "file to write; default is a temp file named after the device"}, {Name: "actor", Default: "cli", Description: "audit actor"}}}, func(ctx cliapp.RunContext) error {
			return captureVerb(ctx, core, "screenshot", 0)
		}),
		command("record", "Record the screen of a TV, Google TV or Android phone for a few seconds, wake it first, save the MP4 and print its path", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id or unique name"}}, Flags: []cliapp.Flag{{Name: "seconds", Default: "5", Description: "duration, 1 to 30 seconds"}, {Name: "home", Bool: true, Description: "press Home first"}, {Name: "no-wake", Bool: true, Description: "do not wake the screen first"}, {Name: "out", Description: "file to write; default is a temp file named after the device"}, {Name: "actor", Default: "cli", Description: "audit actor"}}}, func(ctx cliapp.RunContext) error {
			seconds, err := strconv.Atoi(ctx.Flag("seconds"))
			if err != nil || seconds < 1 || seconds > 30 {
				return fmt.Errorf("seconds must be between 1 and 30")
			}
			return captureVerb(ctx, core, "recording", seconds)
		}),
		command("open", "Open an app such as Netflix or YouTube on a TV, Google TV or Android phone and verify it reached the foreground", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id or unique name"}}, Flags: []cliapp.Flag{{Name: "app", Required: true, Description: "app name (netflix, youtube, prime video, disney+, hulu, max, plex, spotify, settings, chrome) or exact Android package"}, {Name: "no-wake", Bool: true, Description: "do not wake the screen first"}, {Name: "actor", Default: "cli", Description: "audit actor"}}}, func(ctx cliapp.RunContext) error {
			body := map[string]any{"app": ctx.Flag("app"), "actor": ctx.Flag("actor"), "wake": !ctx.BoolFlag("no-wake")}
			raw, err := core.Request(http.MethodPost, "/devices/"+url.PathEscape(ctx.Positional("id"))+"/launch", nil, mustJSON(body))
			if err != nil {
				return err
			}
			if err := emit(ctx, raw, "App launch"); err != nil {
				return err
			}
			var result struct {
				Status      string   `json:"status"`
				Detail      string   `json:"detail"`
				NextActions []string `json:"next_actions"`
			}
			_ = json.Unmarshal(raw, &result)
			if result.Status != "foreground_verified" {
				return fmt.Errorf("app launch %s: %s%s", result.Status, result.Detail, nextActionsSuffix(result.NextActions))
			}
			return nil
		}),
		command("install", "Install an APK file on an Android phone and verify it by package state; --open launches it, --screenshot also captures its screen", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id or unique name"}}, Flags: []cliapp.Flag{{Name: "apk", Required: true, Description: "absolute path of the .apk to install"}, {Name: "open", Bool: true, Description: "open the app and verify it reached the foreground"}, {Name: "screenshot", Bool: true, Description: "open the app and save a screenshot of it"}, {Name: "out", Description: "file for the screenshot; default is a temp file"}, {Name: "actor", Default: "cli", Description: "audit actor"}}}, func(ctx cliapp.RunContext) error {
			apk, err := filepath.Abs(ctx.Flag("apk"))
			if err != nil {
				return err
			}
			body := map[string]any{"apk": apk, "actor": ctx.Flag("actor"), "open": ctx.BoolFlag("open"), "screenshot": ctx.BoolFlag("screenshot")}
			raw, err := core.Request(http.MethodPost, "/devices/"+url.PathEscape(ctx.Positional("id"))+"/install", nil, mustJSON(body))
			if err != nil {
				return err
			}
			var result struct {
				Status      string   `json:"status"`
				Device      string   `json:"device_id"`
				Detail      string   `json:"detail"`
				EvidenceIDs []string `json:"evidence_ids"`
				NextActions []string `json:"next_actions"`
			}
			_ = json.Unmarshal(raw, &result)
			report := map[string]any{}
			_ = json.Unmarshal(raw, &report)
			if len(result.EvidenceIDs) > 0 {
				path, size, dlErr := downloadEvidence(core, result.EvidenceIDs[0], ctx.Flag("out"), strings.ReplaceAll(result.Device, ":", "-"), "screenshot")
				if dlErr != nil {
					return dlErr
				}
				report["path"], report["bytes"] = path, size
			}
			if err := emitValue(ctx, report, "App install"); err != nil {
				return err
			}
			if result.Status != "installed_verified" {
				return fmt.Errorf("install %s: %s%s", result.Status, result.Detail, nextActionsSuffix(result.NextActions))
			}
			return nil
		}),
		command("evidence", "Save a retained screenshot, recording or log by evidence id (as printed by screenshot, record or capture)", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "evidence-id", Required: true, Description: "evidence id"}}, Flags: []cliapp.Flag{{Name: "out", Description: "file to write; default is a temp file"}}}, func(ctx cliapp.RunContext) error {
			path, size, err := downloadEvidence(core, ctx.Positional("evidence-id"), ctx.Flag("out"), "", "")
			if err != nil {
				return err
			}
			return emitValue(ctx, map[string]any{"evidence_id": ctx.Positional("evidence-id"), "path": path, "bytes": size}, "Evidence saved")
		}),
		command("relation-confirm", "Confirm a same-host relation candidate (from device relations) as one physical device, so its ADB transport serves screenshots, recording and app launch", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "candidate id from device relations"}}, Flags: []cliapp.Flag{{Name: "force", Bool: true, Description: "assert the identities are one device although their model names differ"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/devices/relations/confirm", map[string]any{"candidate_id": ctx.Positional("candidate-id"), "force": ctx.BoolFlag("force")}, "Relation confirmed")
		}),
	}
}

// captureVerb runs capture with wake, saves the first evidence artifact and
// reports the saved path with the capture disposition. A capability gap or
// failure prints its next actions and exits non-zero.
func captureVerb(ctx cliapp.RunContext, core *cliapp.ScenarioApp, kind string, seconds int) error {
	id := ctx.Positional("id")
	body := map[string]any{"kind": kind, "actor": ctx.Flag("actor"), "wake": !ctx.BoolFlag("no-wake") || ctx.BoolFlag("home"), "home": ctx.BoolFlag("home")}
	if seconds > 0 {
		body["seconds"] = seconds
	}
	raw, err := core.Request(http.MethodPost, "/devices/"+url.PathEscape(id)+"/capture", nil, mustJSON(body))
	if err != nil {
		return err
	}
	var result struct {
		Disposition string `json:"disposition"`
		Binding     struct {
			DeviceID string `json:"device_id"`
		} `json:"binding"`
		Evidence []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"evidence"`
		Chapters []struct {
			ID, Message, FailureClass string
		} `json:"chapters"`
		NextActions []string `json:"next_actions"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("decode capture result: %w", err)
	}
	report := map[string]any{"device_id": result.Binding.DeviceID, "disposition": result.Disposition}
	if len(result.NextActions) > 0 {
		report["next_actions"] = result.NextActions
	}
	if len(result.Evidence) == 0 {
		if len(result.Chapters) > 0 {
			last := result.Chapters[len(result.Chapters)-1]
			report["message"], report["failure_class"] = last.Message, last.FailureClass
		}
		if err := emitValue(ctx, report, "Capture"); err != nil {
			return err
		}
		return fmt.Errorf("no %s was captured (%s)%s", kind, result.Disposition, nextActionsSuffix(result.NextActions))
	}
	evidenceID := result.Evidence[0].ID
	path, size, err := downloadEvidence(core, evidenceID, ctx.Flag("out"), strings.ReplaceAll(result.Binding.DeviceID, ":", "-"), kind)
	if err != nil {
		return err
	}
	report["evidence_id"], report["path"], report["bytes"] = evidenceID, path, size
	return emitValue(ctx, report, "Capture")
}

// downloadEvidence streams one evidence artifact to out, or to a temp file when
// out is empty. label and kind only shape the default file name.
func downloadEvidence(core *cliapp.ScenarioApp, evidenceID, out, label, kind string) (string, int64, error) {
	if out == "" {
		ext := ".bin"
		switch kind {
		case "screenshot":
			ext = ".png"
		case "recording":
			ext = ".mp4"
		case "logs":
			ext = ".txt"
		}
		dir := filepath.Join(os.TempDir(), "device-control-evidence")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", 0, fmt.Errorf("create evidence directory: %w", err)
		}
		name := strings.TrimLeft(label+"-"+time.Now().UTC().Format("20060102T150405"), "-")
		out = filepath.Join(dir, name+ext)
	}
	file, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("open %s: %w", out, err)
	}
	size, err := core.APIClient.DownloadTo(context.Background(), core.APIPath("/evidence/"+url.PathEscape(evidenceID)), file)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(out)
		return "", 0, fmt.Errorf("download evidence %s: %w", evidenceID, err)
	}
	return out, size, nil
}

func nextActionsSuffix(actions []string) string {
	if len(actions) == 0 {
		return ""
	}
	return "; next: " + strings.Join(actions, " | ")
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}

func emitValue(ctx cliapp.RunContext, value any, title string) error {
	return emit(ctx, mustJSON(value), title)
}
