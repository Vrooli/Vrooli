package agentharness

import (
	"os"
	"path/filepath"
	"strings"
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

// hardStopMatch is what a guard recognized: its class and a short, plain
// description of the signal, which the agent, the block log and the steward
// see. A signal never holds a secret value.
type hardStopMatch struct {
	Class  HardStopClass
	Signal string
}

// hardStopForEvent reports the class of the guard an event trips, or "".
func hardStopForEvent(event ToolEvent) HardStopClass {
	return detectHardStop(event).Class
}

// detectHardStop uses exact hook-visible tokens, never substring matches, and
// judges the acting text of a command rather than the words its data carries.
func detectHardStop(event ToolEvent) hardStopMatch {
	if hardStopsErr != nil {
		return hardStopMatch{}
	}
	stop := func(class HardStopClass, signal string) hardStopMatch {
		return hardStopMatch{Class: class, Signal: signal}
	}
	if header, target, ok := literalHereDocFileWrite(event); ok {
		// A cat heredoc is a local file write: its body is data. Keep explicit
		// writes to a credential path and to a network endpoint protected.
		if isShellNetworkEndpoint(target) || isNetworkRedirection(event) {
			return stop(HardStopOutsideMessage, "writes to a network address")
		}
		if hasSensitiveResource(hookEventTokens(ToolEvent{Target: event.Target, Shell: header})) {
			return stop(HardStopSecret, "writes into a credential file")
		}
		if !ordinaryLocalFileTarget(target, event.WorkingDirectory) {
			return stop(HardStopUnknownEffect, "writes a here-document to a device or special file")
		}
		return hardStopMatch{}
	}
	if isNetworkRedirection(event) {
		return stop(HardStopOutsideMessage, "writes to a network address (/dev/tcp or /dev/udp)")
	}
	acting := event
	if hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), hardStops.Shell.Tools...) {
		acting.Shell, _ = visibleShell(event.Shell)
	}
	tokens := hookEventTokens(acting)
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	if hasSensitiveOperationTarget(event) {
		return stop(HardStopSecret, "reads or writes a credential file")
	}
	if readsProcessEnvironment(acting.Shell) {
		return stop(HardStopSecret, "reads another process's environment (/proc/*/environ)")
	}
	// Search tools report query terms and paths without performing the
	// protected operation described by those terms.
	if isSearchTool(event.Tool) || isNonEffectShellCommand(acting) {
		if hasSensitiveFilePath(hookEventTokens(ToolEvent{Target: event.Target})) && isReadLikeOperation(event) ||
			isNonEffectShellCommand(acting) && hasHookToken(tokens, "grep", "rg", "ripgrep", "ag", "ack", "findstr") && hasSensitiveFilePath(tokens) {
			return stop(HardStopSecret, "searches inside a credential file")
		}
		return hardStopMatch{}
	}
	activeEffect := hasProtectedEffectSemantics(event)
	if isProviderCommentMutation(event.Tool) {
		return stop(HardStopOutsideMessage, "posts a GitHub or GitLab comment")
	}
	secrets, money, messages, hidden := hardStops.Secrets, hardStops.Money, hardStops.Messages, hardStops.HiddenEffect
	// Paired signals (a reading program and a credential path, a chat tool
	// and "send") must meet inside one simple command: `head x; touch y/.env`
	// neither reads a credential nor sends anything.
	commands := actingCommandTokens(event, acting, tokens)
	inOneCommand := func(match func([]string) bool) bool {
		for _, commandTokens := range commands {
			if match(commandTokens) {
				return true
			}
		}
		return false
	}
	readsCredential := func(t []string) bool {
		operation := hasHookToken(t, secrets.Operations...) || hasQualifiedPrefix(t, "process", "env") || hasQualifiedPrefix(t, "os", "environ")
		return operation && (hasSensitiveResource(t) || hasQualifiedCall(t, secrets.Calls...))
	}
	if isVaultReadCommand(event) || hasHookToken(toolTokens, secrets.Tools...) && hasHookToken(toolTokens, secrets.ToolOperations...) {
		return stop(HardStopSecret, "reads from a secret store")
	}
	if activeEffect && inOneCommand(readsCredential) {
		return stop(HardStopSecret, "reads or sends a credential")
	}
	checkout := inOneCommand(func(t []string) bool {
		return hasHookToken(t, money.CheckoutWords...) && hasHookToken(t, money.CheckoutContext...)
	})
	if activeEffect && (hasHookToken(tokens, money.Words...) || checkout || hasQualifiedCall(tokens, money.Calls...)) {
		return stop(HardStopMoney, "moves money (a payment, charge, purchase, invoice or refund)")
	}
	if activeEffect {
		chat := inOneCommand(func(t []string) bool {
			return hasHookToken(t, messages.Transports...) && hasHookToken(t, messages.Verbs...)
		})
		if chat || hasHookToken(tokens, messages.Commands...) {
			return stop(HardStopOutsideMessage, "sends a chat or email message")
		}
		if sequence := publishingCommand(acting); sequence != "" {
			return stop(HardStopOutsideMessage, "publishes outside Vrooli (`"+sequence+"`)")
		}
		if hasHookToken(tokens, messages.Calls...) {
			return stop(HardStopOutsideMessage, "publishes outside Vrooli")
		}
		if host := messageHost(event); host != "" && sendsWebData(event, acting, tokens) {
			return stop(HardStopOutsideMessage, "sends data to a messaging service ("+host+")")
		}
	}
	// Hidden commands: text decoded and then run. A shell tool counts the
	// programs its command line starts, never the tool's own name.
	decodes := hasHookToken(tokens, hidden.EncodedCommandFlags...)
	if hasHookToken(toolTokens, hardStops.Shell.Tools...) {
		bases := shellCommandBases(acting.Shell)
		decodes = decodes || hasHookToken(bases, hidden.Decoders...) && runsShellOrInterpreter(bases)
	}
	if activeEffect && decodes {
		return stop(HardStopUnknownEffect, "decodes text and runs it as a command")
	}
	return hardStopMatch{}
}

// actingCommandTokens splits a shell event's acting text into its simple
// commands and tokenizes each; any other event is one command.
func actingCommandTokens(event, acting ToolEvent, tokens []string) [][]string {
	if !hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), hardStops.Shell.Tools...) || acting.Shell == "" {
		return [][]string{tokens}
	}
	var commands [][]string
	for _, line := range strings.Split(acting.Shell, "\n") {
		for _, command := range splitShellCommands(line) {
			commands = append(commands, hookEventTokens(ToolEvent{Shell: command, Target: event.Target}))
		}
	}
	return commands
}

// runsShellOrInterpreter reports whether a command line starts a shell or an
// interpreter that can run what an earlier stage decoded.
func runsShellOrInterpreter(bases []string) bool {
	for _, base := range bases {
		if isOneOf(base, hardStops.Shell.ShellPrograms) || isInlineInterpreter(base) || base == "eval" {
			return true
		}
	}
	return false
}

// publishingCommand returns the first configured publishing command sequence
// (such as "gh pr create") the acting text runs, or "".
func publishingCommand(acting ToolEvent) string {
	words := strings.Fields(strings.ToLower(strings.Join(append([]string{acting.Tool, acting.Shell}, acting.Arguments...), " ")))
	for index := range words {
		words[index] = strings.Trim(words[index], "'\"`;&|()")
	}
	for _, sequence := range hardStops.Messages.CommandSequences {
		want := strings.Fields(sequence)
		for start := 0; start+len(want) <= len(words); start++ {
			if slicesEqual(words[start:start+len(want)], want) {
				return sequence
			}
		}
	}
	return ""
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

// messageHost returns the configured messaging host the event names, or "".
// The whole event text counts here, literals included, because a URL is data
// the request is about to act on.
func messageHost(event ToolEvent) string {
	text := strings.ToLower(strings.Join(append([]string{event.Tool, event.Shell, event.Target}, event.Arguments...), " "))
	for _, host := range hardStops.Messages.MessageHosts {
		for from := 0; ; {
			index := strings.Index(text[from:], host)
			if index < 0 {
				break
			}
			start, end := from+index, from+index+len(host)
			before, after := byte(' '), byte(' ')
			if start > 0 {
				before = text[start-1]
			}
			if end < len(text) {
				after = text[end]
			}
			if !isHostByte(after) && (before == '.' || !isHostByte(before)) {
				return host
			}
			from = end
		}
	}
	return ""
}

func isHostByte(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '.'
}

// sendsWebData reports a web client (a program or an SDK) used with a body or
// a write method. The client must be in the acting text: a script that only
// edits text naming requests.post is not sending anything. The method may be a
// literal, as in fetch(url, {method: 'POST'}).
func sendsWebData(event, acting ToolEvent, tokens []string) bool {
	messages := hardStops.Messages
	raw := hookEventTokens(event)
	if !hasHookToken(tokens, messages.WebClients...) && !hasQualifiedPrefixAny(tokens, messages.WebClients) {
		return false
	}
	return hasDataSendFlag(acting.Shell) || hasHookToken(raw, messages.WriteMethods...) || hasQualifiedCall(raw, messages.WriteMethods...)
}

func hasQualifiedPrefixAny(tokens []string, prefixes []string) bool {
	for _, token := range tokens {
		head, _, ok := strings.Cut(token, ".")
		if ok && isOneOf(head, prefixes) {
			return true
		}
	}
	return false
}

func hasSensitiveOperationTarget(event ToolEvent) bool {
	readOrWrite := isSearchTool(event.Tool) || hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), hardStops.Secrets.FileTools...)
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
	return isOneOf(base, hardStops.Shell.ShellPrograms)
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
	if !hasHookToken(hookEventTokens(ToolEvent{Tool: event.Tool}), hardStops.Shell.Tools...) {
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
	if !hasHookToken(toolTokens, hardStops.Shell.Tools...) {
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
	quotedDelimiter := strings.HasPrefix(delimFields[0], "'") || strings.HasPrefix(delimFields[0], `"`)
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
		if !quotedDelimiter && hereDocHasExpansion(lines[:index]) {
			return "", "", false
		}
		return header, target, true
	}
	return "", "", false
}

func hereDocHasExpansion(lines []string) bool {
	for _, line := range lines {
		if hasUnescapedExpansion(line) {
			return true
		}
	}
	return false
}

func hasUnescapedExpansion(line string) bool {
	escaped := false
	for _, char := range line {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '$' || char == '`' {
			return true
		}
	}
	return false
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
	messages := hardStops.Messages
	return hasHookToken(tokens, messages.CommentProviders...) && hasHookToken(tokens, messages.CommentMutations...) && hasHookToken(tokens, messages.CommentObjects...)
}

func hasProtectedEffectSemantics(event ToolEvent) bool {
	toolTokens := hookEventTokens(ToolEvent{Tool: event.Tool})
	return hasHookToken(toolTokens, hardStops.EffectTools.Words...) || hasQualifiedCall(toolTokens, hardStops.EffectTools.Calls...)
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
	return hasHookToken(tokens, hardStops.Secrets.Resources...)
}

func hasSensitiveFilePath(tokens []string) bool {
	return hasHookToken(tokens, hardStops.Secrets.FilePaths...)
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
