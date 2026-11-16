package agentharness

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (r Runtime) guardModeStore() *GuardModeStore {
	if r.GuardModes != nil {
		return r.GuardModes
	}
	if r.Store == nil {
		return nil
	}
	return NewGuardModeStore(r.Store.Dir)
}

func (r Runtime) applyLogOnly(event ToolEvent, risk RiskClass, now time.Time, decision *Decision) {
	store := r.guardModeStore()
	if store == nil {
		return
	}
	original := decision.Action
	var mode GuardMode
	var active bool
	var err error
	if original == ActionAllow {
		modes, readErr := store.Active(now)
		err = readErr
		for _, candidate := range modes {
			if candidate.Check == risk {
				mode, active = candidate, true
				break
			}
		}
	} else {
		mode, active, err = store.RecordSuppressedAction(risk, event.EventID, original, now)
	}
	if err != nil {
		decision.Degraded = true
		decision.Evidence = append(decision.Evidence, Evidence{Code: "CHECK_MODES_UNAVAILABLE", Message: "temporary check mode could not be confirmed and normal enforcement remains active", Source: "policy-runtime", Severity: "error"})
		return
	}
	if !active {
		return
	}
	if original != ActionAllow {
		decision.Action = ActionAllow
		decision.Reason = "check is in audited log-only mode until " + mode.ExpiresAt.UTC().Format(time.RFC3339) + "; original action was " + string(original)
	}
	decision.Evidence = append(decision.Evidence, Evidence{Code: "CHECK_LOG_ONLY", Message: "enforcement was observed but not applied under a temporary, audited check mode", Source: "policy-runtime", Facts: map[string]string{"check": string(risk), "mode": mode.Mode, "expires_at": mode.ExpiresAt.UTC().Format(time.RFC3339), "original_action": string(original), "event_id": event.EventID}, Severity: "warning"})
}

func (r Runtime) hardStopDecision(event ToolEvent, risk RiskClass, hardStop HardStopClass, now time.Time) (Decision, error) {
	return r.hardStopDecisionValue(event, risk, hardStop, now), nil
}

func (r Runtime) hardStopDecisionValue(event ToolEvent, risk RiskClass, hardStop HardStopClass, now time.Time) Decision {
	decision := Decision{ContractVersion: ContractVersion, EventID: event.EventID, Action: ActionDeny, Risk: risk, Degraded: false, Reason: "hard-stop action requires operator handling; temporary check relaxations were restored", Evidence: []Evidence{}}
	restored, err := r.restoreGuardModes(hardStop, event, now)
	decision.Evidence = append(decision.Evidence, tripwireEvidence(hardStop, restored, err))
	if err != nil {
		decision.Degraded = true
		decision.Reason = "hard-stop action was denied; restoring temporary check modes failed and requires operator review"
	}
	return decision
}

func (r Runtime) restoreGuardModes(hardStop HardStopClass, event ToolEvent, now time.Time) ([]RiskClass, error) {
	store := r.guardModeStore()
	if store == nil {
		return nil, nil
	}
	// RestoreAll reads and clears the active set under one store lock. Avoid a
	// separate Active pre-read so an enable racing with this event cannot slip
	// between inspection and restoration.
	return store.RestoreAll(hardStop, event.EventID, now)
}

func tripwireEvidence(hardStop HardStopClass, restored []RiskClass, err error) Evidence {
	facts := map[string]string{"hard_stop": string(hardStop), "restored_checks": strings.Join(riskClassStrings(restored), ",")}
	message := "a hard-stop event restored all active temporary log-only modes"
	severity := "warning"
	if err != nil {
		facts["restore_error"] = "restore audit/state write failed"
		message = "hard-stop event detected, but restoration could not be confirmed"
		severity = "error"
	}
	return Evidence{Code: "TRIPWIRE_HARD_STOP", Message: message, Source: "policy-runtime", Facts: facts, Severity: severity}
}

func riskClassStrings(risks []RiskClass) []string {
	out := make([]string, len(risks))
	for i, risk := range risks {
		out[i] = string(risk)
	}
	return out
}

// hardStopForEvent uses exact hook-visible tokens, never substring matches.
// It records no command text or secret material in durable state or evidence.
func hardStopForEvent(event ToolEvent) HardStopClass {
	if header, target, ok := literalHereDocFileWrite(event); ok {
		// A cat heredoc is a local file write: its body is data, not an action
		// description to scan for payments, secrets or outbound messages. Keep
		// explicit writes to a known secret path protected. A special or
		// unresolved redirection endpoint remains a hard stop.
		if isShellNetworkEndpoint(target) || isNetworkRedirection(event) {
			return HardStopOutsideMessage
		}
		if hasSensitiveResource(hookEventTokens(ToolEvent{Target: event.Target, Shell: header})) {
			return HardStopSecret
		}
		if !ordinaryLocalFileTarget(target, event.WorkingDirectory) {
			return HardStopUnknownEffect
		}
		return ""
	}
	if isNetworkRedirection(event) {
		return HardStopOutsideMessage
	}
	tokens := hookEventTokens(event)
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	if hasSensitiveOperationTarget(event) {
		return HardStopSecret
	}
	// Search tools report query terms and paths without performing the
	// protected operation described by those terms.
	if isSearchTool(event.Tool) || isNonEffectShellCommand(event) {
		if hasSensitiveFilePath(hookEventTokens(ToolEvent{Target: event.Target})) && isReadLikeOperation(event) {
			return HardStopSecret
		}
		if isNonEffectShellCommand(event) && hasHookToken(tokens, "grep", "rg", "ripgrep", "ag", "ack", "findstr") && hasSensitiveFilePath(tokens) {
			return HardStopSecret
		}
		return ""
	}
	activeEffect := hasProtectedEffectSemantics(event)
	if isProviderCommentMutation(event.Tool) {
		return HardStopOutsideMessage
	}
	secretTool := hasHookToken(toolTokens, "password", "secret", "secrets", "credential", "credentials", "api_key", "api-key", "private_key", "private-key", "access_token", "refresh_token", "secretsmanager", "secretmanager", "vault")
	secretOperation := hasHookToken(tokens, "read", "readfile", "cat", "head", "tail", "less", "more", "open", "get-content", "getcontent", "write", "edit", "put", "post", "send", "source", "printenv", "env", "export", "curl", "wget", "http", "httpie", "fetch", "requests.get", "requests.post", "httpx.get", "httpx.post", "axios.get", "axios.post", "process.env", "os.environ", "getenv")
	secretOperation = secretOperation || hasQualifiedPrefix(tokens, "process", "env") || hasQualifiedPrefix(tokens, "os", "environ")
	if isVaultReadCommand(event) || secretTool && hasHookToken(toolTokens, "read", "readfile", "get", "fetch", "open", "send") ||
		activeEffect && (hasSensitiveResource(tokens) && secretOperation || hasQualifiedCall(tokens, "password", "secret", "secrets", "credential", "credentials", "api_key", "private_key", "access_token", "refresh_token") && secretOperation) {
		return HardStopSecret
	}
	if activeEffect && (hasHookToken(tokens, "payment", "payments", "purchase", "purchases", "checkout", "billing", "invoice", "charge", "charges", "refund", "refunds", "money-transfer", "wire-transfer", "buy-now", "stripe.charge", "stripe.payment_intents.create", "stripe.paymentintents.create") || hasQualifiedCall(tokens, "paymentintents.create", "payment_intents.create", "paymentintents.confirm", "payment_intents.confirm", "charges.create", "charges.update")) {
		return HardStopMoney
	}
	transport := hasHookToken(tokens, "slack", "discord", "telegram", "whatsapp", "email", "smtp", "webhook", "sendgrid", "mailgun", "sendmail", "mutt")
	outbound := hasHookToken(tokens, "send", "reply", "dispatch", "deliver", "post-message", "post_message", "outbound-message", "outbound_message", "comment", "publish")
	if activeEffect && (transport && outbound || hasHookToken(tokens, "send-message", "send_message", "send-email", "send_email", "sendmail", "mail")) {
		return HardStopOutsideMessage
	}
	if activeEffect && hasHookToken(tokens, "git.push", "gh.pr.comment", "gh.issue.comment", "requests.post", "httpx.post", "axios.post", "fetch.post") {
		return HardStopOutsideMessage
	}
	if activeEffect && hasHookToken(tokens, "curl", "wget", "http", "httpie", "requests.post", "httpx.post", "axios.post", "fetch") && hasHookToken(tokens, "-d", "--data", "--data-raw", "--json", "--form", "-f", "post", "put", "patch") {
		return HardStopOutsideMessage
	}
	if activeEffect && hasQualifiedCall(tokens, "send", "reply", "dispatch", "deliver", "post", "put", "patch", "charge", "payment", "purchase", "secret", "password", "credential", "token", "remove", "unlink", "rmtree", "rmdir", "delete") {
		return HardStopUnknownEffect
	}
	indirect := hasHookToken(tokens, "eval", "exec", "base64", "--encodedcommand", "python", "python3", "node", "deno", "ruby", "perl", "pwsh", "powershell", "bash", "sh")
	if activeEffect && indirect && hasHookToken(tokens, "send", "reply", "dispatch", "deliver", "post", "put", "patch", "charge", "payment", "purchase", "secret", "password", "credential", "token", "environ", "process.env", "os.environ") {
		return HardStopUnknownEffect
	}
	if activeEffect && hasHookToken(tokens, "base64", "--encodedcommand") && hasHookToken(tokens, "bash", "sh", "pwsh", "powershell", "python", "python3", "node", "eval", "exec") {
		return HardStopUnknownEffect
	}
	return ""
}

func hasSensitiveOperationTarget(event ToolEvent) bool {
	readOrWrite := isReadLikeOperation(event) || hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), "write", "writefile", "edit", "put", "patch")
	if !readOrWrite {
		return false
	}
	paths := []string{event.Target}
	for key, value := range event.Context {
		if hasHookToken([]string{strings.ToLower(key)}, "path", "file", "filename", "pathname", "target", "uri") {
			paths = append(paths, value)
		}
	}
	for _, path := range paths {
		if hasSensitiveResource(hookEventTokens(ToolEvent{Target: path})) {
			return true
		}
	}
	return false
}

func isVaultReadCommand(event ToolEvent) bool {
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	if hasHookToken(toolTokens, "vault") && hasHookToken(hookEventTokens(ToolEvent{Arguments: event.Arguments}), "read") {
		return true
	}
	if nested := nestedShellScript(event.Tool, event.Arguments); nested != "" && isVaultReadShell(nested, 0) {
		return true
	}
	return isVaultReadShell(event.Shell, 0)
}

func isVaultReadShell(shell string, depth int) bool {
	if depth >= 4 {
		return false
	}
	for _, command := range splitShellCommands(shell) {
		fields := strings.Fields(strings.TrimSpace(command))
		for index := 0; index < len(fields); {
			word := strings.Trim(fields[index], "'\"")
			base := strings.ToLower(filepath.Base(word))
			switch base {
			case "env":
				index++
				for index < len(fields) && (strings.Contains(fields[index], "=") || strings.HasPrefix(fields[index], "-")) {
					index++
				}
				continue
			case "sudo":
				index++
				for index < len(fields) && strings.HasPrefix(fields[index], "-") {
					option := fields[index]
					index++
					if sudoOptionTakesValue(option) && index < len(fields) {
						index++
					}
				}
				continue
			case "command", "exec":
				index++
				continue
			}
			if strings.Contains(word, "=") && !strings.Contains(word, "/") {
				index++
				continue
			}
			if base == "vault" && index+1 < len(fields) && strings.EqualFold(strings.Trim(fields[index+1], "'\""), "read") {
				return true
			}
			break
		}
		if nested := nestedShellScriptFromCommand(command); nested != "" && isVaultReadShell(nested, depth+1) {
			return true
		}
		if strings.EqualFold(filepath.Base(strings.Fields(command)[0]), "eval") &&
			hasHookToken(hookEventTokens(ToolEvent{Shell: command}), "vault") &&
			hasHookToken(hookEventTokens(ToolEvent{Shell: command}), "read") {
			return true
		}
	}
	for _, nested := range shellSubstitutionScripts(shell) {
		if isVaultReadShell(nested, depth+1) {
			return true
		}
	}
	return false
}

func nestedShellScript(tool string, args []string) string {
	base := strings.ToLower(filepath.Base(strings.Trim(tool, "'\"")))
	if !isShellExecutable(base) {
		return ""
	}
	for index := 0; index+1 < len(args); index++ {
		flag := strings.Trim(args[index], "'\"")
		if strings.HasPrefix(flag, "-") && strings.Contains(flag, "c") {
			return strings.Trim(args[index+1], "'\"")
		}
	}
	return ""
}

func nestedShellScriptFromCommand(command string) string {
	return nestedShellScriptFromCommandDepth(command, 0)
}

func nestedShellScriptFromCommandDepth(command string, depth int) string {
	if depth >= 4 {
		return ""
	}
	fields := shellWords(command)
	for index := 0; index < len(fields); {
		base := strings.ToLower(filepath.Base(strings.Trim(fields[index], "'\"")))
		switch base {
		case "sudo":
			index++
			for index < len(fields) && strings.HasPrefix(fields[index], "-") {
				option := fields[index]
				index++
				if sudoOptionTakesValue(option) && index < len(fields) {
					index++
				}
			}
			continue
		case "env":
			index++
			for index < len(fields) {
				option := fields[index]
				if option == "--" {
					index++
					break
				}
				if option == "-S" || option == "--split-string" {
					if index+1 < len(fields) {
						return nestedShellScriptFromCommandDepth(fields[index+1], depth+1)
					}
					return ""
				}
				if strings.HasPrefix(option, "--split-string=") {
					return nestedShellScriptFromCommandDepth(strings.TrimPrefix(option, "--split-string="), depth+1)
				}
				if strings.HasPrefix(option, "-S") && len(option) > 2 {
					return nestedShellScriptFromCommandDepth(strings.TrimPrefix(option, "-S"), depth+1)
				}
				if strings.HasPrefix(option, "-") {
					index++
					if (option == "-u" || option == "--unset" || option == "-C" || option == "--chdir") && index < len(fields) {
						index++
					}
					continue
				}
				if strings.Contains(option, "=") {
					index++
					continue
				}
				break
			}
			continue
		case "command", "exec":
			index++
			continue
		}
		if !isShellExecutable(base) {
			return ""
		}
		for flagIndex := index + 1; flagIndex+1 < len(fields); flagIndex++ {
			flag := fields[flagIndex]
			if strings.HasPrefix(flag, "-") && strings.Contains(flag, "c") {
				return fields[flagIndex+1]
			}
		}
		return ""
	}
	return ""
}

func sudoOptionTakesValue(option string) bool {
	switch option {
	case "-a", "-u", "-g", "-h", "-p", "-C", "-T", "-R", "-D", "-r", "-t", "-U",
		"--auth-type", "--user", "--group", "--host", "--prompt", "--close-from", "--chroot", "--chdir", "--role", "--type", "--command-timeout", "--other-user":
		return true
	default:
		return false
	}
}

func isShellExecutable(base string) bool {
	switch base {
	case "sh", "bash", "zsh", "dash", "ksh":
		return true
	default:
		return false
	}
}

func shellWords(command string) []string {
	runes := []rune(command)
	var fields []string
	for index := 0; index < len(runes); {
		for index < len(runes) && (runes[index] == ' ' || runes[index] == '\t' || runes[index] == '\r' || runes[index] == '\n') {
			index++
		}
		if index >= len(runes) || runes[index] == ';' || runes[index] == '|' || runes[index] == '&' {
			break
		}
		word, after := shellWordAt(runes, index)
		if after <= index {
			break
		}
		fields = append(fields, word)
		index = after
	}
	return fields
}

func shellSubstitutionScripts(shell string) []string {
	runes := []rune(shell)
	var scripts []string
	quote := rune(0)
	escaped, comment, atWordStart := false, false, true
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if comment {
			if char == '\n' {
				comment, atWordStart = false, true
			}
			continue
		}
		if quote == '\'' {
			if char == '\'' {
				quote = 0
				atWordStart = false
			}
			continue
		}
		if escaped {
			escaped, atWordStart = false, false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote == '"' {
			if char == '"' {
				quote = 0
				atWordStart = false
				continue
			}
			if char == '`' {
				end := index + 1
				for end < len(runes) && runes[end] != '`' {
					if runes[end] == '\\' && end+1 < len(runes) {
						end++
					}
					end++
				}
				if end < len(runes) {
					scripts = append(scripts, string(runes[index+1:end]))
					index = end
				}
				continue
			}
			if char == '$' && index+1 < len(runes) && runes[index+1] == '(' {
				if nested, end, ok := shellParenSubstitution(runes, index+2); ok {
					scripts = append(scripts, nested)
					index = end
				}
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			atWordStart = false
			continue
		}
		if char == '#' && atWordStart {
			comment = true
			continue
		}
		if char == '`' {
			end := index + 1
			for end < len(runes) && runes[end] != '`' {
				if runes[end] == '\\' && end+1 < len(runes) {
					end++
				}
				end++
			}
			if end < len(runes) {
				scripts = append(scripts, string(runes[index+1:end]))
				index = end
			}
			atWordStart = false
			continue
		}
		if char == '$' && index+1 < len(runes) && runes[index+1] == '(' {
			if nested, end, ok := shellParenSubstitution(runes, index+2); ok {
				scripts = append(scripts, nested)
				index = end
			}
			atWordStart = false
			continue
		}
		atWordStart = char == ' ' || char == '\t' || char == '\r' || char == '\n' || char == ';' || char == '|' || char == '&'
	}
	return scripts
}

func shellParenSubstitution(runes []rune, start int) (string, int, bool) {
	depth := 1
	quote := rune(0)
	escaped := false
	for index := start; index < len(runes); index++ {
		char := runes[index]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			quote = char
			continue
		}
		if char == '(' {
			depth++
		} else if char == ')' {
			depth--
			if depth == 0 {
				return string(runes[start:index]), index, true
			}
		}
	}
	return "", len(runes), false
}

func splitShellCommands(script string) []string {
	var commands []string
	var current strings.Builder
	var quote rune
	escaped := false
	comment := false
	atWordStart := true
	runes := []rune(script)
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if comment {
			if char == '\n' {
				if command := strings.TrimSpace(current.String()); command != "" {
					commands = append(commands, command)
				}
				current.Reset()
				comment = false
				atWordStart = true
			}
			continue
		}
		if quote == '\'' {
			current.WriteRune(char)
			if char == '\'' {
				quote = 0
			}
			continue
		}
		if escaped {
			current.WriteRune(char)
			escaped = false
			atWordStart = false
			continue
		}
		if char == '\\' {
			current.WriteRune(char)
			escaped = true
			continue
		}
		if quote != 0 {
			current.WriteRune(char)
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			quote = char
			current.WriteRune(char)
			atWordStart = false
			continue
		}
		if char == '#' && atWordStart {
			comment = true
			continue
		}
		if char == ';' || char == '\n' || char == '|' || char == '&' {
			if command := strings.TrimSpace(current.String()); command != "" {
				commands = append(commands, command)
			}
			current.Reset()
			if (char == '|' || char == '&') && index+1 < len(runes) && runes[index+1] == char {
				index++
			}
			atWordStart = true
			continue
		}
		current.WriteRune(char)
		atWordStart = char == ' ' || char == '\t' || char == '\r'
	}
	if command := strings.TrimSpace(current.String()); command != "" {
		commands = append(commands, command)
	}
	return commands
}

func isNetworkRedirection(event ToolEvent) bool {
	if !hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), "shell", "bash", "sh", "zsh") {
		return false
	}
	shell := strings.TrimSpace(event.Shell)
	if newline := strings.IndexByte(shell, '\n'); newline >= 0 && strings.Contains(shell, "<<") {
		shell = shell[:newline]
	}
	return isNetworkRedirectionShell(shell, 0)
}

func isNetworkRedirectionShell(shell string, depth int) bool {
	if depth >= 4 {
		return false
	}
	for _, target := range shellRedirectionTargets(shell) {
		if isShellNetworkEndpoint(target) {
			return true
		}
	}
	for _, command := range splitShellCommands(shell) {
		if nested := nestedShellScriptFromCommand(command); nested != "" && isNetworkRedirectionShell(nested, depth+1) {
			return true
		}
	}
	for _, nested := range shellSubstitutionScripts(shell) {
		if isNetworkRedirectionShell(nested, depth+1) {
			return true
		}
	}
	return false
}

func shellRedirectionTargets(shell string) []string {
	runes := []rune(shell)
	var targets []string
	var quote rune
	escaped, comment, atWordStart := false, false, true
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if comment {
			if char == '\n' {
				comment, atWordStart = false, true
			}
			continue
		}
		if quote == '\'' {
			if char == '\'' {
				quote = 0
			}
			continue
		}
		if escaped {
			escaped, atWordStart = false, false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			quote, atWordStart = char, false
			continue
		}
		if char == '#' && atWordStart {
			comment = true
			continue
		}
		if char == '>' {
			end := index + 1
			if end < len(runes) && runes[end] == '>' {
				end++
			}
			for end < len(runes) && (runes[end] == ' ' || runes[end] == '\t') {
				end++
			}
			target, after := shellWordAt(runes, end)
			if target != "" {
				targets = append(targets, target)
				index = after - 1
			}
			atWordStart = false
			continue
		}
		atWordStart = char == ' ' || char == '\t' || char == '\r' || char == '\n' || char == ';' || char == '|' || char == '&'
	}
	return targets
}

func shellWordAt(runes []rune, start int) (string, int) {
	var word strings.Builder
	var quote rune
	escaped := false
	index := start
	for ; index < len(runes); index++ {
		char := runes[index]
		if escaped {
			word.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				word.WriteRune(char)
			}
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			quote = char
			continue
		}
		if char == ' ' || char == '\t' || char == '\r' || char == '\n' || char == ';' || char == '|' || char == '&' {
			break
		}
		word.WriteRune(char)
	}
	return word.String(), index
}

func isShellNetworkEndpoint(path string) bool {
	path = strings.ToLower(path)
	return strings.HasPrefix(path, "/dev/tcp/") || strings.HasPrefix(path, "/dev/udp/")
}

// literalHereDocFileWrite identifies a plain `cat ... <<DELIM > path` form.
// The heredoc body is literal data; only the header can read from a protected
// source or name a protected destination. Other commands remain fully scanned.
func literalHereDocFileWrite(event ToolEvent) (string, string, bool) {
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	if !hasHookToken(toolTokens, "shell", "bash", "sh", "zsh") {
		return "", "", false
	}
	newline := strings.IndexByte(event.Shell, '\n')
	if newline < 0 {
		return "", "", false
	}
	header := event.Shell[:newline]
	if !strings.Contains(header, "<<") || !strings.Contains(header, ">") || strings.ContainsAny(header, ";|&`$()") {
		return "", "", false
	}
	fields := strings.Fields(header)
	if len(fields) == 0 || !strings.EqualFold(filepath.Base(strings.Trim(fields[0], "'\"")), "cat") {
		return "", "", false
	}
	target, ok := heredocOutputTarget(fields)
	if !ok {
		return "", "", false
	}
	marker := strings.Index(header, "<<") + 2
	delimFields := strings.Fields(strings.TrimSpace(header[marker:]))
	if len(delimFields) == 0 {
		return "", "", false
	}
	delimiter := strings.Trim(delimFields[0], "'\"")
	if delimiter == "" {
		return "", "", false
	}
	lines := strings.Split(event.Shell[newline+1:], "\n")
	for index, line := range lines {
		if line != delimiter {
			continue
		}
		for _, trailing := range lines[index+1:] {
			if strings.TrimSpace(trailing) != "" {
				return "", "", false
			}
		}
		return header, target, true
	}
	return "", "", false
}

func heredocOutputTarget(fields []string) (string, bool) {
	var target string
	for index := 1; index < len(fields); index++ {
		field := fields[index]
		if field == ">" || field == ">>" {
			if index+1 >= len(fields) || target != "" {
				return "", false
			}
			target = strings.Trim(fields[index+1], "'\"")
			index++
			continue
		}
		if strings.HasPrefix(field, ">") || strings.HasPrefix(field, "2>") {
			if target != "" {
				return "", false
			}
			target = strings.TrimLeft(field, "2>")
			target = strings.Trim(target, "'\"")
		}
	}
	return target, target != ""
}

func ordinaryLocalFileTarget(target, workingDirectory string) bool {
	if target == "" || strings.ContainsAny(target, "$`;&|()<>\n\r") || strings.HasPrefix(target, "//") {
		return false
	}
	if strings.Contains(target, ":") {
		return false
	}
	absolute := target
	if !filepath.IsAbs(absolute) {
		base := workingDirectory
		if base == "" {
			var err error
			base, err = os.Getwd()
			if err != nil {
				return false
			}
		}
		absolute = filepath.Join(base, absolute)
	}
	absolute = filepath.Clean(absolute)
	if hasSpecialFilesystemEndpoint(absolute) {
		return false
	}
	info, err := os.Lstat(absolute)
	if err == nil {
		return info.Mode().IsRegular()
	}
	if !os.IsNotExist(err) {
		return false
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil || hasSpecialFilesystemEndpoint(parent) {
		return false
	}
	return true
}

func hasSpecialFilesystemEndpoint(path string) bool {
	path = filepath.Clean(path)
	return path == "/dev" || strings.HasPrefix(path, "/dev/") || path == "/proc" || strings.HasPrefix(path, "/proc/") || path == "/sys" || strings.HasPrefix(path, "/sys/")
}

func isProviderCommentMutation(tool string) bool {
	tokens := hookEventTokens(ToolEvent{Tool: tool})
	provider := hasHookToken(tokens, "github", "gitlab")
	mutation := hasHookToken(tokens, "create", "add", "post", "send", "publish")
	comment := hasHookToken(tokens, "comment", "note")
	return provider && mutation && comment
}

func hasProtectedEffectSemantics(event ToolEvent) bool {
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	if hasHookToken(toolTokens, "shell", "bash", "sh", "zsh", "pwsh", "powershell", "cmd", "command", "exec", "execute", "python", "python3", "node", "deno", "ruby", "perl", "curl", "wget", "http", "httpie", "fetch", "requests", "httpx", "axios", "stripe", "paypal", "square", "slack", "discord", "telegram", "whatsapp", "email", "smtp", "webhook", "sendgrid", "mailgun", "sendmail", "mutt", "payment", "purchase", "checkout", "billing") {
		return true
	}
	return hasQualifiedCall(toolTokens, "send", "reply", "dispatch", "deliver", "post", "put", "patch", "charge", "payment", "purchase", "create", "confirm", "remove", "unlink", "delete")
}

func isNonEffectShellCommand(event ToolEvent) bool {
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	if !hasHookToken(toolTokens, "shell", "bash", "sh", "zsh", "pwsh", "powershell", "cmd", "command") {
		return false
	}
	// Only trust a single simple display/search/comment command. A compound
	// shell expression must still be inspected for later protected effects.
	if strings.ContainsAny(event.Shell, ";|&\n\r`$()") {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(event.Shell))
	if len(fields) == 0 {
		return false
	}
	command := strings.ToLower(filepath.Base(strings.Trim(fields[0], "'\"`")))
	return hasHookToken([]string{command}, "#", "echo", "printf", "grep", "rg", "ripgrep", "ag", "ack", "findstr")
}

func isSearchTool(tool string) bool {
	return hasHookToken(hookEventTokens(ToolEvent{Tool: tool}), "search", "grep", "rg", "ripgrep", "code-search", "code_search", "search-code")
}

func hasSensitiveResource(tokens []string) bool {
	return hasHookToken(tokens, ".env", ".env.local", ".env.production", ".ssh", "id_rsa", "id_ed25519", ".npmrc", ".netrc", "private_key", "private-key", "api_key", "api-key", "access_token", "refresh_token")
}

func hasSensitiveFilePath(tokens []string) bool {
	return hasHookToken(tokens, ".env", ".env.local", ".env.production", ".ssh", "id_rsa", "id_ed25519", ".npmrc", ".netrc")
}

func isReadLikeOperation(event ToolEvent) bool {
	return isSearchTool(event.Tool) || hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), "read", "readfile", "open", "get-content", "getcontent", "grep", "rg", "ripgrep", "ag", "ack", "findstr")
}

// hasQualifiedCall recognizes a full dotted member name such as client.send.
// It deliberately does not use substring matching, so filenames such as
// send-helper_test.go and secrets_test.go do not become protected effects.
func hasQualifiedCall(tokens []string, actions ...string) bool {
	wanted := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		wanted[action] = struct{}{}
	}
	for _, token := range tokens {
		parts := strings.Split(token, ".")
		if len(parts) < 2 || parts[0] == "" {
			continue
		}
		if _, ok := wanted[parts[len(parts)-1]]; ok {
			return true
		}
	}
	return false
}

func hasQualifiedPrefix(tokens []string, prefix ...string) bool {
	for _, token := range tokens {
		parts := strings.Split(token, ".")
		if len(parts) <= len(prefix) {
			continue
		}
		match := true
		for index, expected := range prefix {
			if parts[index] != expected {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func hookEventTokens(event ToolEvent) []string {
	fields := make([]string, 0, len(event.Arguments)+2+len(event.Context)*2)
	fields = append(fields, event.Shell, event.Target)
	fields = append(fields, event.Arguments...)
	for key, value := range event.Context {
		fields = append(fields, key, value)
	}
	var tokens []string
	appendFieldTokens := func(field string, expandAdapterID bool) {
		for _, token := range strings.FieldsFunc(strings.ToLower(field), func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\'' || r == '"' || r == '`' || r == '/' || r == '\\' || r == ':' || r == '=' || r == ';' || r == '|' || r == '&' || r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' || r == '<' || r == '>' || r == ','
		}) {
			if token != "" {
				tokens = append(tokens, token)
				// MCP/tool adapter IDs commonly encode provider and action as
				// mcp__provider__action_name. Preserve exact tokens and expose
				// components for classification without splitting ordinary
				// identifiers or filenames on underscores.
				if expandAdapterID && strings.Contains(token, "__") {
					for _, component := range strings.Split(token, "__") {
						if component == "" {
							continue
						}
						tokens = append(tokens, strings.Split(component, "_")...)
					}
				}
			}
		}
	}
	appendFieldTokens(event.Tool, true)
	for _, field := range fields {
		appendFieldTokens(field, false)
	}
	return tokens
}

func hasHookToken(tokens []string, candidates ...string) bool {
	wanted := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		wanted[candidate] = struct{}{}
	}
	for _, token := range tokens {
		if _, ok := wanted[token]; ok {
			return true
		}
	}
	return false
}

func outsideRepositoryDeletion(ctx RemovalContext, targets []RemovalTarget) bool {
	if len(targets) == 0 {
		return false
	}
	if strings.TrimSpace(ctx.RepoRoot) == "" {
		return true
	}
	zones := ctx.zones()
	if zones.repo == "" {
		return true
	}
	for _, target := range targets {
		if target.Unresolved != "" {
			return true
		}
		path := zones.targetPath(target)
		if !pathWithinRoot(path, zones.repo, ctx.CaseInsensitive) {
			return true
		}
	}
	return false
}

func pathWithinRoot(path, root string, foldCase bool) bool {
	path, root = filepath.Clean(path), filepath.Clean(root)
	if foldCase {
		path, root = strings.ToLower(path), strings.ToLower(root)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
