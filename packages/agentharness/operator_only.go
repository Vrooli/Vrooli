package agentharness

import (
	"os"
	"strings"
)

// RiskOperatorOnly marks a command that is an operator decision by policy:
// crossing a root control-plane state-contract change or changing automatic
// root update enrollment. It is floor policy, decided the same way under every
// rollout profile and without providers, like filesystem removal.
const RiskOperatorOnly RiskClass = "operator_only"

// operatorOnlyCommand names why a command (its resolved name and argument
// texts) is operator-only, or returns "".
func operatorOnlyCommand(name string, args []string) string {
	switch name {
	case "make", "gmake":
		for _, arg := range args {
			if arg == "rebaseline" {
				return "`make rebaseline` makes a build that changes the root control plane's persistent-state contract the new baseline without automatic rollback; that is the operator's decision"
			}
		}
	case "vrooli":
		return rootUpdateOperatorCommand(args)
	case "vrooli-watchdog":
		return watchdogOperatorCommand(args)
	case "go":
		// go run ./cmd/vrooli runtime update ... reaches the same command.
		for i, arg := range args {
			if path := strings.TrimSuffix(arg, "/"); (path == "cmd/vrooli" || strings.HasSuffix(path, "/cmd/vrooli")) && i > 0 {
				return rootUpdateOperatorCommand(args[i+1:])
			}
		}
	}
	return ""
}

// rootUpdateOperatorCommand matches `runtime update <verb>` for the verbs the
// CLI manifest marks run_eligible=false.
func rootUpdateOperatorCommand(args []string) string {
	words := make([]string, 0, len(args))
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			words = append(words, arg)
		}
	}
	if len(words) < 3 || words[0] != "runtime" || words[1] != "update" {
		return ""
	}
	switch words[2] {
	case "rebaseline":
		return "`vrooli runtime update rebaseline` crosses a root persistent-state contract change without automatic rollback; that is the operator's decision"
	case "enroll", "disable":
		return "`vrooli runtime update " + words[2] + "` changes automatic root update enrollment; that is the operator's decision"
	case "retry":
		return "`vrooli runtime update retry` overrides a failed root release's suppression; that is the operator's decision"
	}
	return ""
}

// watchdogOperatorCommand matches the installed watchdog's own enrollment and
// retry modes, which the CLI verbs forward to.
func watchdogOperatorCommand(args []string) string {
	for _, arg := range args {
		mode, _, _ := strings.Cut(arg, "=")
		switch mode {
		case "--enroll-core-updates", "--disable-core-updates":
			return "`vrooli-watchdog " + mode + "` changes automatic root update enrollment; that is the operator's decision"
		case "--retry-core-update":
			return "`vrooli-watchdog --retry-core-update` overrides a failed root release's suppression; that is the operator's decision"
		}
	}
	return ""
}

// dynamicOperatorCommand decides a command whose name is only known when it
// runs (`$(command -v vrooli)`, an unset variable). It fails closed: arguments
// that spell an operator-only command under any of the names it could be are
// asked about.
func dynamicOperatorCommand(args []string) string {
	for _, name := range []string{"vrooli", "vrooli-watchdog", "make"} {
		if reason := operatorOnlyCommand(name, args); reason != "" {
			return reason
		}
	}
	return ""
}

// relaunchers run their trailing words as a command somewhere the walker does
// not follow as a local shell: a terminal multiplexer pane, a transient
// systemd unit, a repeated or parallel run, a remote host, a command fed from
// standard input. For the operator-only floor every trailing suffix is read as
// command text, so `tmux send-keys -t ops 'make rebaseline' Enter` and
// `systemd-run --user --scope make rebaseline` ask. The scan never feeds the
// removal floor: a remote `rm` is not a local deletion.
var relaunchers = map[string]bool{
	"tmux": true, "screen": true, "systemd-run": true, "watch": true,
	"ssh": true, "parallel": true, "xargs": true,
}

func (w *removalWalker) relaunchedOperatorCommands(name string, args []shellWord) {
	if !relaunchers[name] || w.depth >= maxShellNesting {
		return
	}
	for k := range args {
		if strings.HasPrefix(literalWordText(args[k]), "-") {
			continue
		}
		w.operatorScan(w.joined(args[k:]))
	}
	if name == "xargs" {
		if reason := w.xargsOperatorCommand(args); reason != "" {
			*w.operator = append(*w.operator, reason)
		}
	}
}

// operatorScan walks command text for operator-only commands alone.
func (w *removalWalker) operatorScan(text string) {
	targets := []RemovalTarget{}
	invoked := false
	child := w.child()
	child.depth++
	child.targets, child.invoked = &targets, &invoked
	child.script(text)
}

// xargsOperatorCommand asks when xargs completes an operator-only command
// from standard input, which the hook cannot see: `echo rebaseline | xargs
// make`, `xargs vrooli runtime update`.
func (w *removalWalker) xargsOperatorCommand(args []shellWord) string {
	j, replace := 0, ""
	for j < len(args) {
		t := literalWordText(args[j])
		if !strings.HasPrefix(t, "-") {
			break
		}
		switch t {
		case "-I", "-i", "--replace":
			if j+1 < len(args) {
				replace = literalWordText(args[j+1])
			}
			j++
		case "-a", "-d", "-E", "-L", "-n", "-P", "-s", "--arg-file", "--delimiter", "--max-args", "--max-procs", "--max-chars", "--max-lines", "--process-slot-var", "--eof":
			j++
		}
		if value, ok := strings.CutPrefix(t, "-I"); ok && value != "" {
			replace = value
		}
		j++
	}
	if j >= len(args) {
		return ""
	}
	name, rest := w.commandName(args[j]), w.argTexts(args[j+1:])
	switch name {
	case "make", "gmake":
		if !makeNamesTargets(rest, replace) {
			return "`xargs make` takes its targets from standard input, which can name `rebaseline`; run it from the operator's terminal"
		}
	case "vrooli":
		words := []string{}
		for _, arg := range rest {
			if !strings.HasPrefix(arg, "-") && arg != replace {
				words = append(words, arg)
			}
		}
		if len(words) < 3 && (len(words) == 0 || words[0] == "runtime") && (len(words) < 2 || words[1] == "update") {
			return "`xargs vrooli` completes `runtime update` from standard input, which can name an operator-only verb; run it from the operator's terminal"
		}
	case "vrooli-watchdog":
		return "`xargs vrooli-watchdog` takes its mode from standard input, which can name an enrollment or retry mode; run it from the operator's terminal"
	}
	return ""
}

// makeNamesTargets reports whether make's own arguments name a target, so
// standard input cannot supply one.
func makeNamesTargets(args []string, replace string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-C" || arg == "-f" || arg == "-I" || arg == "-o" || arg == "-W" || arg == "--file" || arg == "--directory" || arg == "--include-dir":
			i++
		case strings.HasPrefix(arg, "-") || strings.Contains(arg, "=") || (replace != "" && strings.Contains(arg, replace)):
		default:
			return true
		}
	}
	return false
}

// operatorEnv is what the walker needs to resolve `~`, `$HOME` and other
// variables in command paths: the host's home and environment, or the
// runtime's removal resolver when one is configured.
func (r Runtime) operatorEnv(event ToolEvent) removalEnv {
	if r.Removal != nil {
		ctx := r.Removal(event.WorkingDirectory)
		return removalEnv{WorkingDirectory: ctx.WorkingDirectory, Home: ctx.Home, Lookup: ctx.Lookup}
	}
	env := removalEnv{WorkingDirectory: event.WorkingDirectory, Lookup: os.LookupEnv}
	if home, err := os.UserHomeDir(); err == nil {
		env.Home = home
	}
	return env
}

// operatorOnlyReasons walks the command text with the removal walker's shell
// handling (wrappers, `bash -c`, `eval`, nested interpreters).
func operatorOnlyReasons(event ToolEvent, env removalEnv) []string {
	var w *removalWalker
	switch {
	case strings.TrimSpace(event.Shell) != "":
		dialect := dialectPOSIX
		if event.Context != nil {
			dialect = dialectFromName(event.Context["shell_dialect"])
		}
		w = newRemovalWalker(env, dialect)
		w.script(event.Shell)
	case strings.TrimSpace(event.Tool) != "" || len(event.Arguments) > 0:
		argv := event.Arguments
		if strings.TrimSpace(event.Tool) != "" {
			argv = append([]string{event.Tool}, event.Arguments...)
		}
		words := make([]shellWord, 0, len(argv))
		for _, arg := range argv {
			words = append(words, quotedWord(arg))
		}
		w = newRemovalWalker(env, dialectPOSIX)
		w.command(words, nil, false)
	default:
		return nil
	}
	return *w.operator
}

// operatorOnlyDecision asks for the operator: an attended harness shows the
// prompt to the person at the keyboard; an unattended one denies, so the
// operator runs the command from their own terminal.
func operatorOnlyDecision(event ToolEvent, reasons []string) Decision {
	return Decision{
		ContractVersion: ContractVersion,
		EventID:         event.EventID,
		Action:          ActionAsk,
		Risk:            RiskOperatorOnly,
		Reason:          reasons[0],
		Evidence:        []Evidence{{Code: "OPERATOR_ONLY_COMMAND", Message: reasons[0], Source: "policy-runtime", Severity: "error"}},
	}
}
