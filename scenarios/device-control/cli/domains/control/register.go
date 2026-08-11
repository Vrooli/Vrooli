package control

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func Group(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "device", Description: "Inventory and onboard governed devices", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("list", "List devices with probed capability snapshots", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/api/v1/devices", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Devices")
		}),
		command("connect", "Show the guided onboarding ladder", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind", Required: true, Description: "android or ios"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/api/v1/devices/connect", map[string]string{"kind": ctx.Flag("kind")}, "Onboarding")
		}),
	}}
}

func StrategyGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "strategy", Description: "Inspect and verify control strategies", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("list", "List every strategy and its probed disposition", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodGet, "/api/v1/strategies", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Strategies")
		}),
		command("verify", "Run the fixed conformance suite", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "strategy id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodGet, "/api/v1/strategies/"+ctx.Positional("id")+"/verify", nil, nil)
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
			b, e := core.Request(http.MethodGet, "/api/v1/sessions", nil, nil)
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
			return post(ctx, core, "/api/v1/sessions/acquire", map[string]any{"device_id": ctx.Flag("device"), "actor": ctx.Flag("actor"), "ttl_seconds": ttl}, "Session")
		}),
		command("kill", "Immediately kill a live session", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "session id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/api/v1/sessions/"+ctx.Positional("id")+"/kill", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Session killed")
		}),
		command("release", "Release a live session", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "session id"}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/api/v1/sessions/"+ctx.Positional("id")+"/release", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Session released")
		}),
	}}
}

func FlowGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "flow", Description: "Validate and run bounded device flows", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("validate", "Validate a flow before taking a lease", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "flow JSON file"}, {Name: "strategy", Required: true, Description: "strategy id"}}}, func(ctx cliapp.RunContext) error { return flowRequest(ctx, core, "/api/v1/flows/validate", false) }),
		command("run", "Run a flow with chaptered evidence", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "flow JSON file"}, {Name: "device", Required: true, Description: "device id"}, {Name: "actor", Default: "cli", Description: "audit actor"}}}, func(ctx cliapp.RunContext) error { return flowRequest(ctx, core, "/api/v1/flows/run", true) }),
	}}
}

func AuditGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "audit", Description: "Review device verb audit records", NeedsAPI: true, Subcommands: []cliapp.Command{command("list", "List audit records", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
		b, e := core.Request(http.MethodGet, "/api/v1/evidence/audit", nil, nil)
		if e != nil {
			return e
		}
		return emit(ctx, b, "Audit")
	})}}
}

func AgentGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "agent", Description: "Run deterministic skill-gated device agents", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("start", "Start an agent run; refuses without the prompt-manager skill", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "goal", Required: true, Description: "goal"}, {Name: "device", Required: true, Description: "device id"}, {Name: "actor", Default: "cli", Description: "audit actor"}, {Name: "skill-available", Bool: true, Description: "confirm the prompt-manager skill is installed"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/api/v1/agents/start", map[string]any{"goal": ctx.Flag("goal"), "device_id": ctx.Flag("device"), "actor": ctx.Flag("actor"), "skill_available": ctx.Flag("skill-available") == "true"}, "Agent")
		}),
		command("abort", "Abort an active agent run", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/api/v1/agents/"+ctx.Positional("id")+"/abort", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Agent aborted")
		}),
		command("promote", "Promote a passing, evidence-backed agent run", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			b, e := core.Request(http.MethodPost, "/api/v1/agents/"+ctx.Positional("id")+"/promote", nil, nil)
			if e != nil {
				return e
			}
			return emit(ctx, b, "Agent promoted")
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
	if run {
		body["device_id"] = ctx.Flag("device")
		body["actor"] = ctx.Flag("actor")
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
