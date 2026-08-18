package control

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

func Group(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "device", Description: "Inventory and onboard governed devices", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("list", "List devices with probed capability snapshots", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/devices", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Devices")
		}),
		command("describe", "Describe one device and every transport capability profile", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/devices/"+ctx.Positional("id"), nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Device description")
		}),
		command("state", "Read live state from a device", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/devices/"+ctx.Positional("id")+"/state", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Device state")
		}),
		command("discover", "Browse the LAN for DNS-SD device services", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "service", Description: "service type to include; repeatable"}, {Name: "timeout-seconds", Default: "10", Description: "bounded browse window"}}}, func(ctx cliapp.RunContext) error {
			timeout, err := strconv.Atoi(ctx.Flag("timeout-seconds"))
			if err != nil || timeout < 1 || timeout > 30 {
				return fmt.Errorf("timeout-seconds must be between 1 and 30")
			}
			query := url.Values{"timeout_seconds": []string{strconv.Itoa(timeout)}}
			for _, service := range ctx.FlagValues("service") {
				if strings.TrimSpace(service) != "" {
					query.Add("service", service)
				}
			}
			b, err := core.Request(http.MethodGet, "/devices/discover", query, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "LAN devices")
		}),
		command("pair", "Pair a Google TV Android TV Remote transport", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}, Flags: []cliapp.Flag{{Name: "pin", Description: "six-character hexadecimal code shown on the television"}, {Name: "pin-stdin", Bool: true, Description: "read the six-character hexadecimal code from standard input"}}}, func(ctx cliapp.RunContext) error {
			deviceID := ctx.Positional("id")
			pin := ctx.Flag("pin")
			if ctx.BoolFlag("pin-stdin") {
				if pin != "" {
					return fmt.Errorf("use either --pin or --pin-stdin, not both")
				}
				return pairInteractive(ctx, core, deviceID, os.Stdin, os.Stderr)
			} else if pin == "" {
				return fmt.Errorf("one of --pin or --pin-stdin is required")
			}
			if len(pin) != 6 || strings.Trim(pin, "0123456789abcdefABCDEF") != "" {
				return fmt.Errorf("pairing code must contain exactly six hexadecimal characters")
			}
			defer func() { pin = "" }()
			return post(ctx, core, "/devices/"+deviceID+"/pair", map[string]string{"pin": pin}, "Device paired")
		}),
		command("merge", "Merge two device identities under an owner-asserted hardware claim", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "canonical device id"}, {Name: "member", Required: true, Description: "device identity to merge"}}, Flags: []cliapp.Flag{{Name: "claim", Required: true, Description: "claim such as cast-id=receiver-id"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/devices/"+ctx.Positional("id")+"/merge", map[string]string{"member_id": ctx.Positional("member"), "claim": ctx.Flag("claim")}, "Device identities merged")
		}),
		command("split", "Split a previously merged device identity", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "merged canonical device id"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/devices/"+ctx.Positional("id")+"/split", nil, "Device identity split")
		}),
		command("actuate", "Send one direct lease-owned device command", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}, Flags: []cliapp.Flag{{Name: "key", Description: "key name such as DPAD_DOWN"}, {Name: "text", Description: "text input"}, {Name: "media", Description: "media action"}, {Name: "property", Description: "property name"}, {Name: "value", Description: "property or media value"}, {Name: "transport", Description: "transport profile"}, {Name: "lease", Required: true, Description: "held lease token"}, {Name: "actor", Default: "cli", Description: "audit actor"}, {Name: "repeat", Default: "1", Description: "bounded repeat count for a key"}}}, func(ctx cliapp.RunContext) error {
			repeat, err := strconv.Atoi(ctx.Flag("repeat"))
			if err != nil || repeat < 1 || repeat > 10 {
				return fmt.Errorf("repeat must be between 1 and 10")
			}
			body := map[string]any{"actor": ctx.Flag("actor"), "lease_token": ctx.Flag("lease")}
			for _, key := range []string{"key", "text", "media", "property", "value", "transport"} {
				if value := ctx.Flag(key); value != "" {
					body[key] = value
				}
			}
			body["repeat"] = repeat
			return post(ctx, core, "/devices/"+ctx.Positional("id")+"/actuate", body, "Device actuation")
		}),
		command("watch", "Print state changes until interrupted", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}}, func(ctx cliapp.RunContext) error {
			return watchDevice(ctx, core, ctx.Positional("id"))
		}),
		command("connect", "Show the guided onboarding ladder", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind", Required: true, Description: "android, ios, or google-tv"}, {Name: "watch", Bool: true, Description: "re-probe until all onboarding rungs are available or the watch window expires"}, {Name: "watch-seconds", Default: "30", Description: "maximum live re-probe window when --watch is set"}}}, func(ctx cliapp.RunContext) error {
			return connect(ctx, core)
		}),
		command("forget", "Forget a retained device identity", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodDelete, "/devices/"+ctx.Positional("id"), nil, nil)
			if e != nil {
				return e
			}
			if len(b) == 0 {
				b, _ = json.Marshal(map[string]any{"device_id": ctx.Positional("id"), "forgotten": true})
			}
			return emit(ctx, b, "Device forgotten")
		}),
		command("promote", "Promote an onboarded Android device to wireless ADB", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "USB-onboarded device id"}}, Flags: []cliapp.Flag{{Name: "transport", Required: true, Description: "wireless"}}}, func(ctx cliapp.RunContext) error {
			if ctx.Flag("transport") != "wireless" {
				return fmt.Errorf("transport must be wireless")
			}
			return post(ctx, core, "/devices/"+ctx.Positional("id")+"/promote", map[string]string{"transport": "wireless"}, "Device transport promotion")
		}),
		command("reconnect", "Reconnect a promoted Android device over wireless ADB", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "device id"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/devices/"+ctx.Positional("id")+"/reconnect", nil, "Wireless device reconnect")
		}),
	}}
}

func readPairingPIN(reader io.Reader, prompt io.Writer) (string, error) {
	if _, err := fmt.Fprint(prompt, "Enter the six-character hexadecimal pairing code shown on the television, then press Enter: "); err != nil {
		return "", fmt.Errorf("write PIN prompt: %w", err)
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64), 64)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read PIN from stdin: %w", err)
		}
		return "", fmt.Errorf("read PIN from stdin: no PIN provided")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func pairInteractive(ctx cliapp.RunContext, core *cliapp.ScenarioApp, deviceID string, reader io.Reader, prompt io.Writer) error {
	startBody, err := core.Request(http.MethodPost, "/devices/"+deviceID+"/pair/start", nil, []byte(`{}`))
	if err != nil {
		return err
	}
	var started struct {
		PairingID string `json:"pairing_id"`
	}
	if err := json.Unmarshal(startBody, &started); err != nil || strings.TrimSpace(started.PairingID) == "" {
		return fmt.Errorf("pairing start returned no session")
	}
	pin, err := readPairingPIN(reader, prompt)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"pairing_id": started.PairingID, "pin": pin})
	if err != nil {
		return fmt.Errorf("encode pairing completion: %w", err)
	}
	response, err := core.Request(http.MethodPost, "/devices/"+deviceID+"/pair/complete", nil, body)
	for i := range body {
		body[i] = 0
	}
	if err != nil {
		return err
	}
	return emit(ctx, response, "Device paired")
}

func StrategyGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "strategy", Description: "Inspect and verify control strategies", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("list", "List every strategy and its probed disposition", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodGet, "/strategies", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Strategies")
		}),
		command("verify", "Run the fixed conformance suite", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "strategy id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodGet, "/strategies/"+ctx.Positional("id")+"/verify", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Conformance")
		}),
	}}
}

func SessionGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "session", Description: "Manage exclusive device leases", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("list", "List live sessions and lease expiry", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodGet, "/sessions", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Sessions")
		}),
		command("acquire", "Acquire an exclusive device lease", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "device", Required: true, Description: "device id"}, {Name: "actor", Required: true, Description: "audit actor"}, {Name: "ttl-seconds", Default: "300", Description: "lease duration"}}}, func(ctx cliapp.RunContext) error {
			ttl, err := strconv.Atoi(ctx.Flag("ttl-seconds"))
			if err != nil || ttl <= 0 {
				return fmt.Errorf("ttl-seconds must be a positive integer")
			}
			return post(ctx, core, "/sessions/acquire", map[string]any{"device_id": ctx.Flag("device"), "actor": ctx.Flag("actor"), "ttl_seconds": ttl}, "Session")
		}),
		command("kill", "Immediately kill a live session", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "session id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/sessions/"+ctx.Positional("id")+"/kill", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Session killed")
		}),
		command("release", "Release a live session", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "session id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/sessions/"+ctx.Positional("id")+"/release", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Session released")
		}),
	}}
}

func FlowGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "flow", Description: "Validate and run bounded device flows", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("validate", "Validate a flow before taking a lease", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "flow JSON file"}, {Name: "strategy", Required: true, Description: "strategy id"}}}, func(ctx cliapp.RunContext) error { return flowRequest(ctx, core, "/flows/validate", false) }),
		command("run", "Run a flow with chaptered evidence", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "flow JSON file"}, {Name: "device", Required: true, Description: "device id"}, {Name: "actor", Default: "cli", Description: "audit actor"}, {Name: "lease", Description: "held lease token"}, {Name: "transport", Description: "usb (default) or wireless; wireless must be explicit"}}}, func(ctx cliapp.RunContext) error { return flowRequest(ctx, core, "/flows/run", true) }),
		command("export", "Export a completed run as a replayable flow", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "completed run id"}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/flows/"+ctx.Positional("run-id")+"/export", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Flow export")
		}),
	}}
}

func AuditGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "audit", Description: "Review device verb audit records", NeedsAPI: true, Subcommands: []cliapp.Command{command("list", "List audit records", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
		b, e := core.Request(http.MethodGet, "/evidence/audit", nil, nil)
		if e != nil {
			return e
		}
		return emit(ctx, b, "Audit")
	})}}
}

func AgentGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "agent", Description: "Run deterministic skill-gated device agents", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("start", "Start an agent run; refuses without the prompt-manager skill", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "goal", Required: true, Description: "goal"}, {Name: "device", Required: true, Description: "device id"}, {Name: "actor", Default: "cli", Description: "audit actor"}, {Name: "skill-available", Bool: true, Description: "confirm the prompt-manager skill is installed"}, {Name: "dry-run", Bool: true, Description: "plan without actuating"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/agents/start", map[string]any{"goal": ctx.Flag("goal"), "device_id": ctx.Flag("device"), "actor": ctx.Flag("actor"), "skill_available": ctx.BoolFlag("skill-available"), "dry_run": ctx.BoolFlag("dry-run")}, "Agent")
		}),
		command("abort", "Abort an active agent run", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/agents/"+ctx.Positional("id")+"/abort", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Agent aborted")
		}),
		command("promote", "Promote a passing, evidence-backed agent run", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/agents/"+ctx.Positional("id")+"/promote", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Agent promoted")
		}),
	}}
}

func ConformanceGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "conformance", Description: "Run governed physical-device conformance plans", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("plan", "Show the Android device capability self-test chapters", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodGet, "/conformance/android", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Android conformance plan")
		}),
		command("run", "Run the Android device capability self-test", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "device", Required: true, Description: "device id"}, {Name: "actor", Default: "cli", Description: "audit actor"}, {Name: "lease", Description: "held lease token"}}}, func(ctx cliapp.RunContext) error {
			body := map[string]any{"device_id": ctx.Flag("device"), "actor": ctx.Flag("actor")}
			if lease := ctx.Flag("lease"); lease != "" {
				body["lease_token"] = lease
			}
			return post(ctx, core, "/conformance/android/run", body, "Android conformance")
		}),
	}}
}

func command(name, description string, args cliapp.ArgSchema, run func(cliapp.RunContext) error) cliapp.Command {
	return cliapp.Command{Name: name, Description: description, NeedsAPI: true, Args: args, RunCtx: run}
}

func post(ctx cliapp.RunContext, core *cliapp.ScenarioApp, path string, value any, title string) error {
	body, _ := json.Marshal(value)
	b, e := core.Request(http.MethodPost, path, nil, body)
	if e != nil {
		return e
	}
	return emit(ctx, b, title)
}

func put(ctx cliapp.RunContext, core *cliapp.ScenarioApp, path string, value any, title string) error {
	body, _ := json.Marshal(value)
	b, e := core.Request(http.MethodPut, path, nil, body)
	if e != nil {
		return e
	}
	return emit(ctx, b, title)
}

const onboardingProbeInterval = 100 * time.Millisecond

func connect(ctx cliapp.RunContext, core *cliapp.ScenarioApp) error {
	kind := ctx.Flag("kind")
	if !ctx.BoolFlag("watch") {
		return post(ctx, core, "/devices/connect", map[string]string{"kind": kind}, "Onboarding")
	}

	seconds, err := strconv.Atoi(ctx.Flag("watch-seconds"))
	if err != nil || seconds <= 0 {
		return fmt.Errorf("watch-seconds must be a positive integer")
	}

	deadline := time.NewTimer(time.Duration(seconds) * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(onboardingProbeInterval)
	defer ticker.Stop()
	previous := map[string]string{}
	transitions := make([]map[string]string, 0)

	for {
		body, requestErr := core.Request(http.MethodPost, "/devices/connect", nil, map[string]string{"kind": kind})
		if requestErr != nil {
			return requestErr
		}
		current := onboardingStatuses(body)
		for id, status := range current {
			if old, ok := previous[id]; ok && old != status {
				transitions = append(transitions, map[string]string{"rung": id, "from": old, "to": status})
			}
		}
		previous = current
		if onboardingReady(body) {
			return emit(ctx, addOnboardingTransitions(body, transitions), "Onboarding")
		}
		select {
		case <-deadline.C:
			return emit(ctx, addOnboardingTransitions(body, transitions), "Onboarding")
		case <-ticker.C:
		}
	}
}

func onboardingStatuses(body []byte) map[string]string {
	var report struct {
		Rungs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"rungs"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return map[string]string{}
	}
	statuses := make(map[string]string, len(report.Rungs))
	for _, rung := range report.Rungs {
		statuses[rung.ID] = rung.Status
	}
	return statuses
}

func addOnboardingTransitions(body []byte, transitions []map[string]string) []byte {
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		return body
	}
	value["transitions"] = transitions
	out, err := json.Marshal(value)
	if err != nil {
		return body
	}
	return out
}

func onboardingReady(body []byte) bool {
	var report struct {
		Rungs []struct {
			Status string `json:"status"`
		} `json:"rungs"`
	}
	if err := json.Unmarshal(body, &report); err != nil || len(report.Rungs) == 0 {
		return false
	}
	for _, rung := range report.Rungs {
		if rung.Status != "available" {
			return false
		}
	}
	return true
}

func flowRequest(ctx cliapp.RunContext, core *cliapp.ScenarioApp, path string, run bool) error {
	raw, e := os.ReadFile(ctx.Flag("file"))
	if e != nil {
		return fmt.Errorf("read flow: %w", e)
	}
	var flow json.RawMessage
	if e = json.Unmarshal(raw, &flow); e != nil {
		return fmt.Errorf("parse flow: %w", e)
	}
	body := map[string]any{"flow": json.RawMessage(flow)}
	if run && ctx.Flag("transport") != "" {
		var definition map[string]any
		if e = json.Unmarshal(raw, &definition); e != nil {
			return fmt.Errorf("parse flow definition: %w", e)
		}
		definition["transport"] = ctx.Flag("transport")
		flow, e = json.Marshal(definition)
		if e != nil {
			return fmt.Errorf("encode flow definition: %w", e)
		}
		body["flow"] = json.RawMessage(flow)
	}
	if run {
		body["device_id"] = ctx.Flag("device")
		body["actor"] = ctx.Flag("actor")
		if lease := ctx.Flag("lease"); lease != "" {
			body["lease_token"] = lease
		}
	} else {
		body["strategy_id"] = ctx.Flag("strategy")
	}
	return post(ctx, core, path, body, "Flow")
}

func watchDevice(ctx cliapp.RunContext, core *cliapp.ScenarioApp, deviceID string) error {
	watchCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	endpoint := strings.TrimRight(core.APIBase(), "/") + core.APIPath("/devices/"+deviceID+"/events")
	req, err := http.NewRequestWithContext(watchCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create watch request: %w", err)
	}
	core.HTTPClient.ApplyRequestHeaders(req)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("open device event stream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("device event stream returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "data:") {
			_, _ = fmt.Fprintln(ctx.Stdout(), strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil && watchCtx.Err() == nil {
		return fmt.Errorf("read device event stream: %w", err)
	}
	return nil
}

func emit(ctx cliapp.RunContext, body []byte, title string) error {
	var value any
	if e := json.Unmarshal(body, &value); e != nil {
		value = map[string]any{"raw": string(body)}
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), value)
	}
	pretty, _ := json.MarshalIndent(value, "", "  ")
	return ctx.RenderList(cliapp.ListReport{Summary: []string{title + " report"}, ResultsHeading: title, Results: []string{string(pretty)}})
}

var _ = strings.TrimSpace
