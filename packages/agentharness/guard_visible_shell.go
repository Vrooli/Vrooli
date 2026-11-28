package agentharness

import (
	"path/filepath"
	"strings"
	"unicode"
)

// The visible-action floor judges what a shell command does, not the words its
// data carries. visibleShell reduces a shell command to the text that acts:
// command words, flags, single-word arguments (paths, URLs, hosts), and code
// handed to an interpreter or a remote shell. It drops data: multi-word quoted
// text such as a commit message, a bug report or a search pattern, and the
// bodies of heredocs written to files. Without this, a bug report that names a
// payment or a grep for "Secret" was refused as if it moved money or read a
// credential.
//
// Every $(...) or backtick substitution is kept, because the shell executes it
// even inside double quotes.

const visibleShellMaxDepth = 4

// visibleShell returns the acting text of shell and whether it hands inline
// code to an interpreter (python -c, a python heredoc), which is opaque in the
// same way a script file is.
func visibleShell(shell string) (string, bool) {
	return visibleShellDepth(shell, 0)
}

type hereDocMarker struct {
	stripTabs bool
	quoted    bool
	delimiter string
	offset    int
}

func visibleShellDepth(shell string, depth int) (string, bool) {
	if depth >= visibleShellMaxDepth {
		// Too deeply nested to reduce: judge it whole, as before.
		return shell, false
	}
	var out []string
	inlineCode := false
	runes := []rune(shell)
	for pos := 0; pos < len(runes); {
		line, next := readLogicalLine(runes, pos)
		pos = next
		text, code := visibleCommandLine(line, depth)
		out = append(out, text)
		inlineCode = inlineCode || code
		for _, marker := range hereDocMarkers(line) {
			var body []string
			for pos < len(runes) {
				end := pos
				for end < len(runes) && runes[end] != '\n' {
					end++
				}
				physical := string(runes[pos:end])
				pos = end + 1
				candidate := physical
				if marker.stripTabs {
					candidate = strings.TrimLeft(candidate, "\t")
				}
				if candidate == marker.delimiter {
					break
				}
				body = append(body, physical)
			}
			text, code := visibleHereDocBody(strings.Join(body, "\n"), hereDocConsumer(line[:marker.offset]), marker.quoted, depth)
			if text != "" {
				out = append(out, text)
			}
			inlineCode = inlineCode || code
		}
	}
	return strings.Join(out, "\n"), inlineCode
}

// readLogicalLine reads up to a newline that ends a command: not inside
// quotes and not escaped. A backslash-newline continuation becomes a space.
func readLogicalLine(runes []rune, pos int) (string, int) {
	var line strings.Builder
	var quote rune
	escaped := false
	for ; pos < len(runes); pos++ {
		char := runes[pos]
		if escaped {
			escaped = false
			line.WriteRune(char)
			continue
		}
		if char == '\\' && quote != '\'' {
			if pos+1 < len(runes) && runes[pos+1] == '\n' {
				line.WriteRune(' ') // line continuation
				pos++
				continue
			}
			escaped = true
			line.WriteRune(char)
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			line.WriteRune(char)
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			line.WriteRune(char)
			continue
		}
		if char == '\n' {
			return line.String(), pos + 1
		}
		line.WriteRune(char)
	}
	return line.String(), pos
}

// hereDocMarkers finds << operators outside quotes on a logical line.
func hereDocMarkers(line string) []hereDocMarker {
	runes := []rune(line)
	var markers []hereDocMarker
	var quote rune
	escaped := false
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if char != '<' || index+1 >= len(runes) || runes[index+1] != '<' {
			continue
		}
		if index+2 < len(runes) && runes[index+2] == '<' {
			index += 2 // here-string
			continue
		}
		marker := hereDocMarker{offset: index}
		at := index + 2
		if at < len(runes) && runes[at] == '-' {
			marker.stripTabs = true
			at++
		}
		for at < len(runes) && (runes[at] == ' ' || runes[at] == '\t') {
			at++
		}
		if at < len(runes) && (runes[at] == '\'' || runes[at] == '"') {
			marker.quoted = true
			at++
		}
		start := at
		for at < len(runes) && (runes[at] == '_' || unicode.IsLetter(runes[at]) || unicode.IsDigit(runes[at])) {
			at++
		}
		if at == start {
			continue
		}
		marker.delimiter = string(runes[start:at])
		markers = append(markers, marker)
		index = at
	}
	return markers
}

// visibleHereDocBody keeps a heredoc body only where it runs: shell code is
// reduced again, interpreter code keeps its code but not its literals, and a
// body fed to anything else is data. An unquoted body still runs its
// substitutions.
func visibleHereDocBody(body, consumer string, quoted bool, depth int) (string, bool) {
	switch {
	case isShellExecutable(consumer):
		return visibleShellDepth(body, depth+1)
	case isInlineInterpreter(consumer):
		return stripCodeLiterals(body), true
	case quoted:
		return "", false
	default:
		return visibleSubstitutions(body, depth)
	}
}

// hereDocConsumer names the command a heredoc feeds: the last simple command
// on the header before the << operator.
func hereDocConsumer(prefix string) string {
	commands := splitShellCommands(prefix)
	if len(commands) == 0 {
		return ""
	}
	words := shellWords(commands[len(commands)-1])
	base, _ := commandBase(words)
	return base
}

// visibleCommandLine reduces one physical line (which may hold several
// commands) word by word.
func visibleCommandLine(line string, depth int) (string, bool) {
	var out []string
	inlineCode := false
	commands := splitShellCommands(line)
	// Text piped into a shell or interpreter that reads its program from stdin
	// is code, not data: judge such a line whole.
	for index, command := range commands {
		words := shellWords(command)
		base, baseIndex := commandBase(words)
		if index > 0 && (isShellExecutable(base) || isInlineInterpreter(base)) && readsProgramFromStdin(words[baseIndex+1:]) {
			return line, true
		}
	}
	for _, command := range commands {
		text, code := visibleSimpleCommand(command, depth)
		out = append(out, text)
		inlineCode = inlineCode || code
	}
	return strings.Join(out, " ; "), inlineCode
}

type visibleWord struct {
	raw      string // as written, quotes included
	value    string // after quote removal
	quoted   bool   // any part was quoted
	double   bool   // a double-quoted part exists (substitutions run there)
	hasSpace bool   // the value holds whitespace: prose, not a path or flag
}

func visibleSimpleCommand(command string, depth int) (string, bool) {
	words := visibleWords(command)
	if len(words) == 0 {
		return "", false
	}
	values := make([]string, len(words))
	for index, word := range words {
		values[index] = word.value
	}
	base, baseIndex := commandBase(values)
	inlineCode := false
	if isPureReader(base) {
		// A reader only displays or searches: its patterns and expressions are
		// data. Keep the program, flags, redirections and paths it reads.
		var out []string
		for index, word := range words {
			if index <= baseIndex || strings.HasPrefix(word.value, "-") || strings.HasPrefix(word.raw, ">") || strings.HasPrefix(word.raw, "<") || strings.HasPrefix(word.raw, "2>") || isPathLike(word.value) {
				out = append(out, word.value)
			}
			if word.double || !word.quoted {
				if text, code := visibleSubstitutions(word.raw, depth); text != "" {
					out = append(out, text)
					inlineCode = inlineCode || code
				}
			}
		}
		return strings.Join(out, " "), inlineCode
	}
	var out []string
	for index, word := range words {
		previous := ""
		if index > 0 {
			previous = values[index-1]
		}
		if !word.quoted {
			out = append(out, word.raw)
			// An unquoted word can still hold a substitution: $(...) runs.
			if strings.Contains(word.raw, "$(") || strings.Contains(word.raw, "`") {
				text, code := visibleSubstitutions(word.raw, depth)
				out = append(out, text)
				inlineCode = inlineCode || code
			}
			continue
		}
		switch {
		case index > baseIndex && isShellExecutable(base) && isShellCodeFlag(previous):
			text, code := visibleShellDepth(word.value, depth+1)
			out = append(out, text)
			inlineCode = inlineCode || code
		case index > baseIndex && isInlineInterpreter(base) && isInterpreterCodeFlag(previous):
			out = append(out, stripCodeLiterals(word.value))
			inlineCode = true
		case index > baseIndex && isRemoteShellCarrier(base):
			text, code := visibleShellDepth(word.value, depth+1)
			out = append(out, text)
			inlineCode = inlineCode || code
		case !word.hasSpace:
			out = append(out, word.value)
		}
		// Multi-word text (a message, a sed script, a search pattern) is data:
		// a path named inside it is not a file the command opens. Its $(...)
		// substitutions still run and are kept below.
		if word.double {
			text, code := visibleSubstitutions(word.raw, depth)
			if text != "" {
				out = append(out, text)
			}
			inlineCode = inlineCode || code
		}
	}
	return strings.Join(out, " "), inlineCode
}

// visibleSubstitutions keeps only the $(...) and backtick scripts of text,
// reduced, since the shell runs them wherever they appear.
func visibleSubstitutions(text string, depth int) (string, bool) {
	var out []string
	inlineCode := false
	for _, script := range shellSubstitutionScripts(strings.ReplaceAll(text, "'", "")) {
		reduced, code := visibleShellDepth(script, depth+1)
		out = append(out, reduced)
		inlineCode = inlineCode || code
	}
	return strings.Join(out, " ; "), inlineCode
}

// isPureReader names programs that only display, search or count their input
// and cannot run another command. find (-exec), awk (system), sed (e) and
// xargs are not readers.
func isPureReader(base string) bool { return isOneOf(base, hardStops.Shell.Readers) }

// isOneOf reports exact membership in a hard_stops.json list.
func isOneOf(value string, list []string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}
	return false
}

func isPathLike(value string) bool {
	return strings.ContainsAny(value, "/~$") || strings.HasPrefix(value, ".")
}

// readsProgramFromStdin reports a shell or interpreter invocation with no
// script argument (or "-"/-s), which runs whatever is piped in.
func readsProgramFromStdin(args []string) bool {
	for _, arg := range args {
		switch {
		case arg == "-" || arg == "-s":
			return true
		case arg == "-c" || isShellCodeFlag(arg) || isInterpreterCodeFlag(arg):
			return false
		case strings.HasPrefix(arg, "<") || strings.HasPrefix(arg, ">") || strings.HasPrefix(arg, "2>"):
			continue
		case strings.HasPrefix(arg, "-"):
			continue
		default:
			return false
		}
	}
	return true
}

func visibleWords(command string) []visibleWord {
	runes := []rune(command)
	var words []visibleWord
	index := 0
	for index < len(runes) {
		for index < len(runes) && unicode.IsSpace(runes[index]) {
			index++
		}
		if index >= len(runes) {
			break
		}
		start := index
		var value strings.Builder
		word := visibleWord{}
		var quote rune
		escaped := false
		for ; index < len(runes); index++ {
			char := runes[index]
			if escaped {
				value.WriteRune(char)
				escaped = false
				continue
			}
			if char == '\\' && quote != '\'' {
				escaped = true
				continue
			}
			if quote != 0 {
				if char == quote {
					quote = 0
					continue
				}
				value.WriteRune(char)
				continue
			}
			if char == '\'' || char == '"' {
				quote = char
				word.quoted = true
				word.double = word.double || char == '"'
				continue
			}
			if char == '$' && index+1 < len(runes) && runes[index+1] == '(' {
				// Keep a substitution inside the word intact.
				if _, end, ok := shellParenSubstitution(runes, index+2); ok {
					value.WriteString(string(runes[index : end+1]))
					index = end
					continue
				}
			}
			if unicode.IsSpace(char) {
				break
			}
			value.WriteRune(char)
		}
		word.raw = string(runes[start:index])
		word.value = value.String()
		word.hasSpace = strings.IndexFunc(word.value, unicode.IsSpace) >= 0
		words = append(words, word)
	}
	return words
}

// commandBase is the program a simple command runs, after variable
// assignments and wrappers such as timeout, env, sudo and nohup.
func commandBase(words []string) (string, int) {
	for index := 0; index < len(words); index++ {
		word := words[index]
		if word == "" || strings.HasPrefix(word, "-") {
			continue
		}
		if eq := strings.IndexByte(word, '='); eq > 0 && !strings.ContainsAny(word[:eq], "/.") {
			continue // VAR=value
		}
		base := strings.ToLower(filepath.Base(word))
		if isOneOf(base, hardStops.Shell.Wrappers) {
			// timeout takes a duration before the command.
			if base == "timeout" && index+1 < len(words) && len(words[index+1]) > 0 && unicode.IsDigit(rune(words[index+1][0])) {
				index++
			}
			continue
		}
		return base, index
	}
	return "", len(words)
}

func isInlineInterpreter(base string) bool {
	return isOneOf(base, hardStops.Shell.Interpreters) || strings.HasPrefix(base, "python3.")
}

func isShellCodeFlag(flag string) bool {
	if flag == "-c" || flag == "--command" {
		return true
	}
	// Combined short flags such as -lc or -ec.
	return strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && strings.HasSuffix(flag, "c")
}

func isInterpreterCodeFlag(flag string) bool {
	return isOneOf(flag, hardStops.Shell.InterpreterCodeFlags)
}

// isRemoteShellCarrier names commands whose quoted arguments are shell code
// run elsewhere or later.
func isRemoteShellCarrier(base string) bool {
	return isOneOf(base, hardStops.Shell.RemoteShellCarriers)
}

// stripCodeLiterals keeps an interpreter program's code and drops its string
// literals and comments, so a script that edits text naming a protected word
// is not read as performing it. Calls, imports and attribute names stay.
func stripCodeLiterals(code string) string {
	runes := []rune(code)
	var out strings.Builder
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		atBoundary := index == 0 || unicode.IsSpace(runes[index-1])
		switch {
		case char == '#' && atBoundary:
			for index < len(runes) && runes[index] != '\n' {
				index++
			}
			out.WriteRune('\n')
		case char == '/' && index+1 < len(runes) && runes[index+1] == '/' && atBoundary:
			for index < len(runes) && runes[index] != '\n' {
				index++
			}
			out.WriteRune('\n')
		case char == '\'' || char == '"' || char == '`':
			triple := index+2 < len(runes) && runes[index+1] == char && runes[index+2] == char
			end := index + 1
			if triple {
				end = index + 3
			}
			for end < len(runes) {
				if runes[end] == '\\' {
					end += 2
					continue
				}
				if triple {
					if runes[end] == char && end+2 < len(runes) && runes[end+1] == char && runes[end+2] == char {
						end += 2
						break
					}
				} else if runes[end] == char || (runes[end] == '\n' && char != '`') {
					break
				}
				end++
			}
			out.WriteString(`""`)
			index = end
		default:
			out.WriteRune(char)
		}
	}
	return out.String()
}

// readsProcessEnvironment reports a read of another process's environment
// (/proc/<pid>/environ), which carries its credentials.
// Only a command that reads the file counts: a reader program given the path,
// or any command with stdin redirected from it. A path named in prose does not.
func readsProcessEnvironment(shell string) bool {
	isEnviron := func(word string) bool {
		word = strings.Trim(word, `'"`)
		return strings.HasPrefix(word, "/proc/") && strings.HasSuffix(word, "/environ")
	}
	for _, line := range strings.Split(shell, "\n") {
		for _, command := range splitShellCommands(line) {
			words := strings.Fields(command)
			base, _ := commandBase(shellWords(command))
			reader := isOneOf(base, hardStops.Shell.EnvironmentReaders)
			for index, word := range words {
				if isEnviron(strings.TrimPrefix(word, "<")) && (reader || strings.HasPrefix(word, "<") || (index > 0 && words[index-1] == "<")) {
					return true
				}
			}
		}
	}
	return false
}

// shellCommandBases lists the program each simple command in shell runs.
func shellCommandBases(shell string) []string {
	var bases []string
	for _, line := range strings.Split(shell, "\n") {
		for _, command := range splitShellCommands(line) {
			if base, _ := commandBase(shellWords(command)); base != "" {
				bases = append(bases, base)
			}
		}
	}
	return bases
}

// hasDataSendFlag reports a curl/wget-style command that sends a body. Flags
// are read only on that command and compared with their case: curl -D saves
// headers and -f fails quietly, and another tool's --json is not curl's.
func hasDataSendFlag(shell string) bool {
	for _, line := range strings.Split(shell, "\n") {
		for _, command := range splitShellCommands(line) {
			// The client may run under a wrapper such as xargs, so look for it
			// anywhere in the command and read the flags that follow it.
			words := shellWords(command)
			for index, word := range words {
				if isOneOf(strings.ToLower(filepath.Base(word)), hardStops.Messages.WebClientPrograms) && hasBodyFlag(words[index+1:]) {
					return true
				}
			}
		}
	}
	return false
}

func hasBodyFlag(fields []string) bool {
	for _, field := range fields {
		if isOneOf(field, hardStops.Messages.BodyFlags) {
			return true
		}
		for _, prefix := range hardStops.Messages.BodyFlagPrefixes {
			if strings.HasPrefix(field, prefix) {
				return true
			}
		}
		if strings.HasPrefix(field, "-d") && len(field) > 2 && !strings.HasPrefix(field, "--") {
			return true // -d@file, -d{...}
		}
	}
	return false
}
