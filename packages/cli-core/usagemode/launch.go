package usagemode

import (
	"strconv"
	"strings"
	"time"
)

// Plan is what a mode changes about one launch. Applied is false (and Args/Env
// are the inputs, untouched) when no mode applies.
type Plan struct {
	Applied bool
	ModeID  string
	// Arm is set for a randomized operator-session day: incumbent or
	// challenger. Label states it visibly for the launcher to print.
	Arm       string
	Exception string
	Label     string
	Args      []string
	// Env holds KEY=VALUE additions only; the caller's environment is not copied.
	Env []string
}

// LaunchPlan computes the overrides for a harness launch. It never fails: any
// problem yields an unchanged plan. The returned Args alias the input when
// nothing changed.
func LaunchPlan(runner string, args, env []string, now time.Time) Plan {
	plan := Plan{Args: args}
	switch NormalizeRunner(runner) {
	case "claude", "codex":
	default:
		return plan
	}
	mode, ok := Resolve(Query{Runner: runner, Now: now})
	if !ok {
		return learnPlan(runner, args, env, now, plan)
	}
	settings, _ := mode.SettingsFor(runner)
	switch NormalizeRunner(runner) {
	case "claude":
		plan.Env = claudeEnv(settings, args, env)
	case "codex":
		plan.Args = codexArgs(settings, args)
	}
	if len(plan.Env) == 0 && len(plan.Args) == len(args) {
		plan.Args = args
		return plan
	}
	plan.Applied, plan.ModeID = true, mode.ID
	plan.Env = append(plan.Env, EnvMode+"="+mode.ID)
	return plan
}

func envHas(env []string, key string) bool {
	for _, entry := range env {
		if name, _, ok := strings.Cut(entry, "="); ok && name == key {
			return true
		}
	}
	return false
}

func claudeEnv(s RunnerSettings, args, env []string) []string {
	for _, arg := range args {
		switch {
		case arg == "--resume", arg == "-r", arg == "--continue", arg == "-c",
			strings.HasPrefix(arg, "--resume="):
			return nil
		// The alias remap would otherwise turn an explicit `--model opus` into
		// the mode's model, and an explicit model always wins.
		case arg == "--model", strings.HasPrefix(arg, "--model="):
			return nil
		}
	}
	// A caller-set model is an explicit choice, so the alias remaps that would
	// rewrite it are skipped too.
	if envHas(env, "ANTHROPIC_MODEL") {
		return nil
	}
	var out []string
	set := func(key, value string) {
		if value != "" && !envHas(env, key) {
			out = append(out, key+"="+value)
		}
	}
	if SessionModelAllowed("claude", s) {
		set("ANTHROPIC_MODEL", s.SessionModel)
	}
	set("ANTHROPIC_DEFAULT_OPUS_MODEL", s.ModelMap["opus"])
	set("ANTHROPIC_DEFAULT_SONNET_MODEL", s.ModelMap["sonnet"])
	// CLAUDE_CODE_SUBAGENT_MODEL is deliberately never set: it forces every
	// sub-agent onto one model and would raise a Haiku sub-agent. Sub-agents
	// inherit the session or follow the alias remap above, which only lowers.
	set("CLAUDE_CODE_EFFORT_LEVEL", s.Effort)
	if limit, _ := ClaudeContextLimit(s.ContextLimitTokens); limit > 0 {
		set("CLAUDE_CODE_AUTO_COMPACT_WINDOW", strconv.Itoa(limit))
	}
	return out
}

// MinClaudeContextLimit is the smallest CLAUDE_CODE_AUTO_COMPACT_WINDOW Claude
// Code honours.
const MinClaudeContextLimit = 100000

// ClaudeContextLimit is the CLAUDE_CODE_AUTO_COMPACT_WINDOW to set for a
// requested limit. Claude ignores values under MinClaudeContextLimit, so a
// smaller request is raised to that minimum (the tightest limit Claude
// honours) and clamped is true so callers can state it. Zero means no limit.
func ClaudeContextLimit(requested int) (limit int, clamped bool) {
	switch {
	case requested <= 0:
		return 0, false
	case requested < MinClaudeContextLimit:
		return MinClaudeContextLimit, true
	}
	return requested, false
}

// codexValueFlags take a separate value argument, so the value is not a subcommand.
var codexValueFlags = map[string]bool{
	"-c": true, "--config": true, "-m": true, "--model": true, "-C": true, "--cd": true,
	"-p": true, "--profile": true, "-s": true, "--sandbox": true, "-a": true,
	"--ask-for-approval": true, "-i": true, "--image": true, "--add-dir": true,
	"--local-provider": true, "--enable": true, "--disable": true,
	"-o": true, "--output-last-message": true, "--output-schema": true, "--color": true,
}

var codexSubcommands = map[string]bool{
	"review": true, "login": true, "logout": true, "mcp": true, "mcp-server": true,
	"app-server": true, "resume": true, "fork": true, "apply": true, "a": true,
	"cloud": true, "completion": true, "sandbox": true, "debug": true, "features": true,
	"proto": true, "help": true, "plugin": true, "remote-control": true,
}

// codexScan is what a Codex argv already states.
type codexScan struct {
	// model is any explicit choice of model or provider: -m, -c model=,
	// -c model_provider=, -p/--profile and --oss each pick the model.
	model    bool
	keys     map[string]bool // -c keys
	sub      string
	resumed  bool // `exec ... resume`
	subLower bool // sub is a non-session subcommand
}

func scanCodex(args []string) codexScan {
	scan := codexScan{keys: map[string]bool{}}
	execSeen := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			name, value, hasValue := strings.Cut(arg, "=")
			switch name {
			case "-m", "--model", "--oss", "-p", "--profile", "--local-provider":
				scan.model = true
			case "-c", "--config":
				if !hasValue && i+1 < len(args) {
					i++
					value = args[i]
				}
				if key, _, ok := strings.Cut(value, "="); ok {
					key = strings.TrimSpace(key)
					scan.keys[key] = true
					if key == "model" || key == "model_provider" {
						scan.model = true
					}
				}
				continue
			}
			if codexValueFlags[name] && !hasValue {
				i++
			}
			continue
		}
		switch {
		case scan.sub == "":
			scan.sub = arg
		case (scan.sub == "exec" || scan.sub == "e") && !execSeen:
			// The first positional after exec is a prompt or `resume`; the
			// values of flags such as -o were skipped above.
			execSeen = true
			scan.resumed = arg == "resume"
		}
	}
	scan.subLower = scan.sub != "" && scan.sub != "exec" && scan.sub != "e" && codexSubcommands[scan.sub]
	return scan
}

func codexArgs(s RunnerSettings, args []string) []string {
	scan := scanCodex(args)
	// Resumed sessions keep their settings. An explicit model (an Agent Manager
	// run always passes one, exempt roles included) means the caller already
	// decided the whole launch, so no effort or context override is added.
	if scan.subLower || scan.resumed || scan.model {
		return args
	}
	var prefix []string
	add := func(key, value string) {
		if value != "" && !scan.keys[key] {
			prefix = append(prefix, "-c", key+"="+value)
		}
	}
	if SessionModelAllowed("codex", s) {
		add("model", s.SessionModel)
	}
	add("model_reasoning_effort", s.Effort)
	if s.ContextLimitTokens > 0 {
		add("model_auto_compact_token_limit", strconv.Itoa(s.ContextLimitTokens))
	}
	if len(prefix) == 0 {
		return args
	}
	return append(prefix, args...)
}

// learnPlan is the launch under an opted-in Learn mode. Whatever the arm, the
// plan carries the mode and arm markers; only a challenger day changes the
// launch. An explicit model or a full-power request keeps the model and is
// recorded as an exception: the session still belongs to its assigned arm.
func learnPlan(runner string, args, env []string, now time.Time, plan Plan) Plan {
	mode, ok := ResolveLearnSession(runner, now)
	if !ok {
		return plan
	}
	plan.Arm = mode.SessionArm(now)
	plan.ModeID = mode.ID
	plan.Env = []string{EnvMode + "=" + mode.ID, EnvArm + "=" + plan.Arm}
	if plan.Arm == ArmChallenger {
		settings, _ := mode.ChallengerSettings(runner)
		var changed []string
		switch NormalizeRunner(runner) {
		case "claude":
			changed = claudeEnv(settings, args, env)
		case "codex":
			plan.Args = codexArgs(settings, args)
		}
		plan.Env = append(plan.Env, changed...)
		switch {
		case envTrue(env, EnvFullPower):
			plan.Exception = ExceptionFullPower
		case explicitModel(runner, args, env):
			plan.Exception = ExceptionExplicitModel
		}
		if plan.Exception != "" {
			plan.Env = append(plan.Env, ArmExceptionKey+"="+plan.Exception)
			plan.Args = args
			plan.Env = dropEnv(plan.Env, changed)
		}
	}
	plan.Applied = true
	plan.Label = "Learn mode " + mode.ID + ": " + plan.Arm + " day"
	if plan.Exception != "" {
		plan.Label += " (" + plan.Exception + " this launch; counted in " + plan.Arm + ")"
	}
	return plan
}

func envTrue(env []string, key string) bool {
	for _, entry := range env {
		if name, value, ok := strings.Cut(entry, "="); ok && name == key {
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "1", "true", "yes", "on":
				return true
			}
		}
	}
	return false
}

func dropEnv(env, drop []string) []string {
	skip := map[string]bool{}
	for _, entry := range drop {
		skip[entry] = true
	}
	var out []string
	for _, entry := range env {
		if !skip[entry] {
			out = append(out, entry)
		}
	}
	return out
}

// explicitModel reports a launch that names its own model, which beats any mode.
func explicitModel(runner string, args, env []string) bool {
	if NormalizeRunner(runner) == "codex" {
		return scanCodex(args).model
	}
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "--model" || strings.HasPrefix(arg, "--model=") {
			return true
		}
	}
	return envHas(env, "ANTHROPIC_MODEL") && NormalizeRunner(runner) == "claude"
}
