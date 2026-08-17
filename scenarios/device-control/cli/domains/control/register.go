package control

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
		command("connect", "Show the guided onboarding ladder", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind", Required: true, Description: "android or ios"}, {Name: "watch", Bool: true, Description: "re-probe until all onboarding rungs are available or the watch window expires"}, {Name: "watch-seconds", Default: "30", Description: "maximum live re-probe window when --watch is set"}}}, func(ctx cliapp.RunContext) error {
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
		command("start", "Start an agent run; refuses without the prompt-manager skill", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "goal", Required: true, Description: "goal"}, {Name: "device", Required: true, Description: "device id"}, {Name: "actor", Default: "cli", Description: "audit actor"}, {Name: "skill-available", Bool: true, Description: "confirm the prompt-manager skill is installed"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/agents/start", map[string]any{"goal": ctx.Flag("goal"), "device_id": ctx.Flag("device"), "actor": ctx.Flag("actor"), "skill_available": ctx.Flag("skill-available") == "true"}, "Agent")
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
