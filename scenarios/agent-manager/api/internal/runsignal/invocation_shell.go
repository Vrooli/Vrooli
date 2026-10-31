package runsignal

import (
	"path/filepath"
	"regexp"
	"strings"
)

// rawSegment is one simple command of a shell line: its words with quoting
// and redirections removed, and the operator that follows it.
type rawSegment struct {
	Tokens []string
	Op     string // "&&", "||", "|", ";", "&" or "" for the last segment
}

type heredoc struct {
	delim string
	strip bool
}

// lexShell is a tolerant, non-evaluating shell lexer. Unlike the strict
// segmenter it understands quotes, heredocs (bodies are skipped), command
// substitutions (kept inside their word), comments, redirections, subshell
// grouping and every list operator, so ordinary agent scripts produce
// command words rather than "unparseable". It reports false only for
// unterminated quotes or substitutions.
func lexShell(s string) ([]rawSegment, bool) {
	var (
		segments []rawSegment
		cur      []string
		word     strings.Builder
		inWord   bool
		pending  []heredoc
	)
	flush := func() {
		if inWord {
			cur = append(cur, word.String())
			word.Reset()
			inWord = false
		}
	}
	endSegment := func(op string) {
		flush()
		if len(cur) > 0 {
			segments = append(segments, rawSegment{Tokens: cur, Op: op})
			cur = nil
		} else if len(segments) > 0 && op != "" && segments[len(segments)-1].Op == ";" {
			segments[len(segments)-1].Op = op
		}
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r':
			flush()
			i++
		case c == '\n':
			endSegment(";")
			i++
			for _, h := range pending {
				i = skipHeredocBody(s, i, h)
			}
			pending = nil
		case c == '\\' && i+1 < len(s) && s[i+1] == '\n':
			i += 2
		case c == '#' && !inWord:
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case (c == '<' || c == '>') && i+1 < len(s) && s[i+1] == '(':
			next, ok := skipExpansion(s, i)
			if !ok {
				return nil, false
			}
			word.WriteString(s[i:next])
			inWord = true
			i = next
		case c == '<' || c == '>' || (c == '&' && i+1 < len(s) && s[i+1] == '>'):
			// A numeric word directly before the operator is a descriptor.
			if inWord && allDigits(word.String()) {
				word.Reset()
				inWord = false
			}
			flush()
			if strings.HasPrefix(s[i:], "<<<") {
				_, next, ok := readShellWord(s, skipBlanks(s, i+3))
				if !ok {
					return nil, false
				}
				i = next
				continue
			}
			if h, next, ok := heredocOperator(s, i); ok {
				pending = append(pending, h)
				i = next
				continue
			}
			j := i
			for j < len(s) && (s[j] == '<' || s[j] == '>' || s[j] == '&' || s[j] == '|') {
				j++
			}
			if j < len(s) && (s[j-1] == '&') && (s[j] >= '0' && s[j] <= '9' || s[j] == '-') {
				for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '-') {
					j++
				}
				i = j
				continue
			}
			_, next, ok := readShellWord(s, skipBlanks(s, j))
			if !ok {
				return nil, false
			}
			i = next
		case c == '&':
			if i+1 < len(s) && s[i+1] == '&' {
				endSegment("&&")
				i += 2
			} else {
				endSegment("&")
				i++
			}
		case c == '|':
			switch {
			case i+1 < len(s) && s[i+1] == '|':
				endSegment("||")
				i += 2
			case i+1 < len(s) && s[i+1] == '&':
				endSegment("|")
				i += 2
			default:
				endSegment("|")
				i++
			}
		case c == ';':
			endSegment(";")
			i++
			for i < len(s) && s[i] == ';' {
				i++
			}
		case c == '(' || c == ')':
			flush()
			i++
		default:
			text, next, ok := readShellWord(s, i)
			if !ok {
				return nil, false
			}
			if next == i { // defensive: never stall
				next = i + 1
			}
			word.WriteString(text)
			inWord = true
			i = next
		}
	}
	endSegment("")
	if len(segments) > 0 {
		segments[len(segments)-1].Op = ""
	}
	return segments, true
}

func skipBlanks(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

// heredocOperator reads `<<WORD`, `<<-WORD`, `<<'WORD'` or `<<"WORD"` at i and
// returns the heredoc and the index after its delimiter word. The delimiter
// is the word with its quotes removed, as the shell reads it; the one parser
// serves both the lexer and a heredoc inside $(...).
func heredocOperator(s string, i int) (heredoc, int, bool) {
	if !strings.HasPrefix(s[i:], "<<") || strings.HasPrefix(s[i:], "<<<") {
		return heredoc{}, i, false
	}
	j := i + 2
	h := heredoc{}
	if j < len(s) && s[j] == '-' {
		h.strip = true
		j++
	}
	delim, next, ok := readShellWord(s, skipBlanks(s, j))
	if !ok || delim == "" {
		return heredoc{}, i, false
	}
	h.delim = delim
	return h, next, true
}

// readShellWord consumes one shell word starting at s[i], returning its
// unquoted text and the index after it. It reports false for an unterminated
// quote or substitution.
func readShellWord(s string, i int) (string, int, bool) {
	var b strings.Builder
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ';' || c == '|' || c == '&' || c == '<' || c == '>' || c == '(' || c == ')':
			return b.String(), i, true
		case c == '\\':
			if i+1 < len(s) && s[i+1] != '\n' {
				b.WriteByte(s[i+1])
			}
			i += 2
		case c == '\'':
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return "", i, false
			}
			b.WriteString(s[i+1 : i+1+end])
			i += end + 2
		case c == '"':
			text, next, ok := scanDoubleQuoted(s, i)
			if !ok {
				return "", i, false
			}
			b.WriteString(text)
			i = next
		case c == '$' || c == '`':
			next, ok := skipExpansion(s, i)
			if !ok {
				return "", i, false
			}
			b.WriteString(s[i:next])
			i = next
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), i, true
}

// skipHeredocBody advances past the lines of a heredoc body, through its
// terminator line.
func skipHeredocBody(s string, i int, h heredoc) int {
	for i < len(s) {
		end := strings.IndexByte(s[i:], '\n')
		line := s[i:]
		next := len(s)
		if end >= 0 {
			line = s[i : i+end]
			next = i + end + 1
		}
		if h.strip {
			line = strings.TrimLeft(line, "\t")
		}
		if strings.TrimRight(line, "\r") == h.delim {
			return next
		}
		i = next
	}
	return i
}

// scanDoubleQuoted returns the text of a double-quoted string beginning at
// s[i] and the index after its closing quote. Expansions inside are kept raw.
func scanDoubleQuoted(s string, i int) (string, int, bool) {
	var b strings.Builder
	for j := i + 1; j < len(s); {
		switch s[j] {
		case '"':
			return b.String(), j + 1, true
		case '\\':
			if j+1 < len(s) {
				if next := s[j+1]; next == '"' || next == '\\' || next == '$' || next == '`' {
					b.WriteByte(next)
				} else {
					b.WriteByte('\\')
					b.WriteByte(next)
				}
			}
			j += 2
		case '$', '`':
			next, ok := skipExpansion(s, j)
			if !ok {
				return "", i, false
			}
			b.WriteString(s[j:next])
			j = next
		default:
			b.WriteByte(s[j])
			j++
		}
	}
	return "", i, false
}

// skipExpansion returns the index after a $(...), ${...}, $'...', $"...",
// `...` or <(...) / >(...) construct starting at s[i].
func skipExpansion(s string, i int) (int, bool) {
	switch s[i] {
	case '`':
		for j := i + 1; j < len(s); j++ {
			if s[j] == '\\' {
				j++
			} else if s[j] == '`' {
				return j + 1, true
			}
		}
		return i, false
	case '$':
		if i+1 >= len(s) {
			return i + 1, true
		}
		switch s[i+1] {
		case '(':
			return skipBalanced(s, i+1, '(', ')')
		case '{':
			return skipBalanced(s, i+1, '{', '}')
		case '\'':
			for j := i + 2; j < len(s); j++ {
				if s[j] == '\\' {
					j++
				} else if s[j] == '\'' {
					return j + 1, true
				}
			}
			return i, false
		}
		return i + 1, true
	}
	// process substitution: s[i] is the < or >.
	return skipBalanced(s, i+1, '(', ')')
}

// skipBalanced skips a bracketed region starting at s[open], aware of quotes.
func skipBalanced(s string, open int, lo, hi byte) (int, bool) {
	depth := 0
	for j := open; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '<':
			// A heredoc inside $(...) has a body whose apostrophes are text.
			if h, after, ok := heredocOperator(s, j); ok {
				if nl := strings.IndexByte(s[after:], '\n'); nl >= 0 {
					j = skipHeredocBody(s, after+nl+1, h) - 1
				}
			}
		case '\'':
			end := strings.IndexByte(s[j+1:], '\'')
			if end < 0 {
				return open, false
			}
			j += end + 1
		case '"':
			_, next, ok := scanDoubleQuoted(s, j)
			if !ok {
				return open, false
			}
			j = next - 1
		case lo:
			depth++
		case hi:
			depth--
			if depth == 0 {
				return j + 1, true
			}
		}
	}
	return open, false
}

var envAssignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// controlPrefixes are shell keywords that precede a command in the same
// segment (`do rg x`, `then git add`); controlOnly keywords carry no command.
var (
	controlPrefixes = map[string]bool{"if": true, "then": true, "else": true, "elif": true, "do": true, "while": true, "until": true, "!": true, "time": true, "{": true, "}": true}
	controlOnly     = map[string]bool{"for": true, "done": true, "fi": true, "esac": true, "case": true, "select": true, "function": true, "in": true}
	noiseCommands   = map[string]bool{"cd": true, "pushd": true, "popd": true, "export": true, "unset": true, "set": true, "shopt": true, "trap": true, "true": true, ":": true, "alias": true, "local": true, "type": true, "which": true, "hash": true}
	shellPrograms   = map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true}
	// noMatchPrograms exit 1 to report "nothing found / differs", not failure.
	noMatchPrograms = map[string]bool{"rg": true, "grep": true, "egrep": true, "fgrep": true, "diff": true, "cmp": true, "test": true, "[": true, "[[": true, "ag": true, "ack": true}
)

// wrapperFlagValues lists, per transparent wrapper, the flags that take a value.
var wrapperFlagValues = map[string]map[string]bool{
	"env":     {"-u": true, "-C": true, "-S": true},
	"timeout": {"-s": true, "-k": true, "--signal": true, "--kill-after": true},
	"sudo":    {"-u": true, "-g": true, "-C": true, "-h": true, "-p": true},
	"nohup":   {},
	"nice":    {"-n": true},
	"ionice":  {"-c": true, "-n": true},
	"command": {},
	"exec":    {"-a": true},
	"builtin": {},
	"stdbuf":  {"-i": true, "-o": true, "-e": true},
	"setsid":  {},
	"time":    {},
}

var durationWord = regexp.MustCompile(`^[0-9.]+[smhd]?$`)

// stripWrappers removes leading environment assignments and transparent
// wrappers (env, timeout, sudo, nohup, nice...) from a command's words.
func stripWrappers(tokens []string) []string {
	for {
		for len(tokens) > 0 && envAssignment.MatchString(tokens[0]) {
			tokens = tokens[1:]
		}
		if len(tokens) == 0 {
			return nil
		}
		name := filepath.Base(tokens[0])
		valueFlags, wrapper := wrapperFlagValues[name]
		if !wrapper {
			return tokens
		}
		if name == "command" && len(tokens) > 1 && (tokens[1] == "-v" || tokens[1] == "-V") {
			// `command -v x` asks whether x exists; it does not run x.
			return nil
		}
		i, sawDuration := 1, false
		for i < len(tokens) {
			t := tokens[i]
			switch {
			case valueFlags[t]:
				i += 2
			case strings.HasPrefix(t, "-"), name == "env" && envAssignment.MatchString(t):
				i++
			case name == "timeout" && !sawDuration && durationWord.MatchString(t):
				i, sawDuration = i+1, true
			default:
				goto stripped
			}
		}
	stripped:
		if i >= len(tokens) {
			return nil
		}
		tokens = tokens[i:]
	}
}

// shellScript returns the script of `bash -lc '<script>'` style words.
func shellScript(tokens []string) (wrapper, script string, ok bool) {
	if len(tokens) < 3 || !shellPrograms[filepath.Base(tokens[0])] {
		return "", "", false
	}
	for i := 1; i < len(tokens)-1; i++ {
		flag := tokens[i]
		if !strings.HasPrefix(flag, "-") || strings.HasPrefix(flag, "--") {
			return "", "", false
		}
		if strings.Contains(flag, "c") {
			return filepath.Base(tokens[0]) + " " + flag, tokens[i+1], true
		}
	}
	return "", "", false
}

// commandUnit is one simple command of a tool call, ready to resolve.
type commandUnit struct {
	Kind       unitKind
	Tokens     []string // program first, program name already path-stripped
	Wrapper    string
	Reason     string
	FromScript bool // read from a script's exec_command call, so it is a shell command whatever the tool is named
	Command    int  // which command of the call it belongs to: the exec_command index, codeCommand for code
	Last       bool // inherits its command's exit status
	Piped      bool // reads another segment's output through a pipe
	AllAnd     bool // every operator in the chain is &&, so success is provable for all
}

type unitKind int

const (
	unitCommand unitKind = iota
	unitNone
	unitCode
	unitUnparseable
)

// shellUnits turns shell text into command units. The bool is false when the
// text cannot be lexed (unterminated quote).
func shellUnits(command string) ([]commandUnit, bool) {
	segments, ok := lexShell(command)
	if !ok {
		return nil, false
	}
	allAnd := true
	for _, seg := range segments {
		if seg.Op != "" && seg.Op != "&&" {
			allAnd = false
		}
	}
	words := make([][]string, len(segments))
	last := -1 // the last segment that runs a command; `done`/`fi` do not
	for i, seg := range segments {
		tokens := seg.Tokens
		for len(tokens) > 0 && controlPrefixes[tokens[0]] {
			tokens = tokens[1:]
		}
		if len(tokens) > 0 && !controlOnly[tokens[0]] {
			if words[i] = stripWrappers(tokens); len(words[i]) > 0 {
				last = i
			}
		}
	}
	var kept, noise []commandUnit
	for i, tokens := range words {
		if len(tokens) == 0 {
			continue
		}
		// A unit after && or || may never have run, so the call's exit
		// status cannot be attributed to it; ; and pipeline tails inherit it.
		afterChain := i > 0 && (segments[i-1].Op == "&&" || segments[i-1].Op == "||")
		unit := commandUnit{Last: i == last && !afterChain, AllAnd: allAnd, Piped: i > 0 && segments[i-1].Op == "|"}
		if wrapper, script, wrapped := shellScript(tokens); wrapped {
			inner, innerOK := shellUnits(script)
			if !innerOK {
				return nil, false
			}
			for j := range inner {
				inner[j].Wrapper = wrapper
				inner[j].Last = inner[j].Last && unit.Last
				inner[j].AllAnd = inner[j].AllAnd && allAnd
			}
			kept = append(kept, inner...)
			continue
		}
		unit.Tokens = append([]string{filepath.Base(tokens[0])}, tokens[1:]...)
		if noiseCommands[unit.Tokens[0]] {
			noise = append(noise, unit)
		} else {
			kept = append(kept, unit)
		}
	}
	if len(kept) == 0 {
		kept = noise
	}
	return kept, true
}

// argvUnits is the literal-argv form: words are never reparsed as syntax,
// except for the `bash -lc` wrapper whose script is shell text.
func argvUnits(args []string) ([]commandUnit, bool) {
	tokens := stripWrappers(args)
	if len(tokens) == 0 {
		return nil, true
	}
	if wrapper, script, wrapped := shellScript(tokens); wrapped {
		units, ok := shellUnits(script)
		for i := range units {
			units[i].Wrapper = wrapper
		}
		return units, ok
	}
	return []commandUnit{{Tokens: append([]string{filepath.Base(tokens[0])}, tokens[1:]...), Last: true, AllAnd: true}}, true
}

// subcommandPrograms are tools whose first word selects the operation, with
// the flags that take a value before it.
var subcommandPrograms = map[string]map[string]bool{
	"git": {"-C": true, "-c": true}, "docker": {"-H": true, "--context": true}, "go": {}, "npm": {}, "pnpm": {}, "yarn": {},
	"cargo": {}, "kubectl": {"-n": true, "--namespace": true, "--context": true}, "systemctl": {}, "gh": {"-R": true, "--repo": true},
	"make": {"-C": true, "-f": true, "-j": true}, "tmux": {"-L": true, "-S": true}, "pip": {}, "apt": {}, "brew": {}, "podman": {}, "helm": {},
}

var subcommandWord = regexp.MustCompile(`^[a-z][a-z0-9:_-]*$`)

// commandPathFor is the stable identity of a command: the longest manifest
// path for catalogued Vrooli CLIs, otherwise the program plus its subcommand
// word when the program is subcommand-style.
func commandPathFor(resolution CatalogResolution, tokens []string) string {
	if resolution.Command != "" && resolution.Command != tokens[0] {
		// A manifest group does not list every operation under it; the next
		// word (`test-genie runs <verb>`) still selects it.
		if resolution.Group {
			if sub := subcommandAfter(tokens, strings.Count(resolution.Command, " ")+1, nil); sub != "" && !strings.Contains(resolution.Command, sub) {
				return resolution.Command + " " + sub
			}
		}
		return resolution.Command
	}
	program := tokens[0]
	valueFlags, ok := subcommandPrograms[program]
	if !ok {
		return program
	}
	if sub := subcommandAfter(tokens, 1, valueFlags); sub != "" {
		return program + " " + sub
	}
	return program
}

// subcommandAfter returns the first non-flag word from tokens[from:] when it
// looks like a subcommand, skipping the given value-taking flags.
func subcommandAfter(tokens []string, from int, valueFlags map[string]bool) string {
	for i := from; i < len(tokens); i++ {
		t := tokens[i]
		switch {
		case valueFlags[t]:
			i++
		case strings.HasPrefix(t, "-"):
		case subcommandWord.MatchString(t):
			return t
		default:
			return ""
		}
	}
	return ""
}
