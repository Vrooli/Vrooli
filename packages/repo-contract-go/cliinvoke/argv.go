package cliinvoke

// The argv catalog. Every supervisor-side invocation of the vrooli CLI is
// built here and nowhere else, so the invoker registry test in
// internal/cli/rootcli/invokers can parse the production argv of each site
// through the real root parser without any site retyping it. None of these
// argvs starts with a global flag: behavior switches travel as VROOLI_*
// environment variables that an older or newer CLI ignores harmlessly.

// ScenarioLifecycle returns `scenario <verb> <name> [--best-effort]`.
func ScenarioLifecycle(verb, scenario string, bestEffort bool) []string {
	argv := []string{"scenario", verb, scenario}
	if bestEffort {
		argv = append(argv, "--best-effort")
	}
	return argv
}

// ScenarioSetup returns `scenario setup <name>`.
func ScenarioSetup(scenario string) []string {
	return []string{"scenario", "setup", scenario}
}

// ScenarioRestartInstance returns `scenario restart <name> --instance <variant>`.
func ScenarioRestartInstance(scenario, variant string) []string {
	return []string{"scenario", "restart", scenario, "--instance", variant}
}

// ScenarioStatusJSON returns `scenario status <name> --json`.
func ScenarioStatusJSON(scenario string) []string {
	return []string{"scenario", "status", scenario, "--json"}
}

// ScenarioPort returns `scenario port <name> <key>`.
func ScenarioPort(scenario, key string) []string {
	return []string{"scenario", "port", scenario, key}
}

// ScenarioPortJSON returns `scenario port <name> <key> --json`.
func ScenarioPortJSON(scenario, key string) []string {
	return append(ScenarioPort(scenario, key), "--json")
}

// RuntimeSupervisorRun returns `runtime supervisor run`, the argv the native
// scheduler unit and the direct launcher share.
func RuntimeSupervisorRun() []string {
	return []string{"runtime", "supervisor", "run"}
}

// AgentRecover returns the argv the autoheal API uses to hand a scenario to
// the recovery broker.
func AgentRecover(scenario, reason, requester string) []string {
	return []string{"agent", "recover", "--scenario", scenario, "--reason", reason, "--requester", requester}
}

// Setup returns `setup` with an optional --json.
func Setup(jsonOutput bool) []string {
	if jsonOutput {
		return []string{"setup", "--json"}
	}
	return []string{"setup"}
}

// SetupStatusReadiness returns the readiness inspection argv the autoheal
// readiness check runs.
func SetupStatusReadiness() []string {
	return []string{"setup", "status", "--json", "--phase", "readiness"}
}

// DiagnosePort returns `diagnose-port <port> [scenario]`.
func DiagnosePort(port, scenario string) []string {
	argv := []string{"diagnose-port", port}
	if scenario != "" {
		argv = append(argv, scenario)
	}
	return argv
}

// VersionJSON returns `version --json`, the loop preflight's first probe.
func VersionJSON() []string {
	return []string{"version", "--json"}
}
