package agentharness

import (
	"errors"
	"fmt"
	"strings"
)

// The removal engine reads command text as data. This lexer turns POSIX,
// PowerShell, and cmd.exe command text into words and operators without ever
// executing it. Expansion is deferred to the walker, because a variable
// assigned earlier in the same command line changes what a later word means.

type shellDialect int

const (
	dialectPOSIX shellDialect = iota
	dialectPowerShell
	dialectCmd
)

// dialectFromName maps the event's declared shell to a lexer dialect. An
// undeclared shell is POSIX: every supported agent's Bash tool runs one, on
// Windows included (Git Bash).
func dialectFromName(name string) shellDialect {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "powershell", "pwsh":
		return dialectPowerShell
	case "cmd", "cmd.exe":
		return dialectCmd
	default:
		return dialectPOSIX
	}
}

type partKind int

const (
	// partText is literal text; quoted text never globs.
	partText partKind = iota
	// partVariable names a variable resolved at walk time.
	partVariable
	// partHome is a leading tilde meaning the user's home directory.
	partHome
	// partDynamic is a value only known when the command runs, such as a
	// command substitution or a positional parameter.
	partDynamic
)

type wordPart struct {
	kind   partKind
	text   string
	quoted bool
}

type shellWord struct {
	parts []wordPart
	raw   string
	// nested holds the command and process substitutions inside the word;
	// they run before the command that contains them.
	nested [][]shellToken
}

type tokenKind int

const (
	tokenWord tokenKind = iota
	tokenOperator
)

type shellToken struct {
	kind tokenKind
	word shellWord
	op   string
	// body is the here-document text attached to a << operator.
	body string
}

// shellOperators is ordered longest first so the lexer matches greedily.
var shellOperators = []string{
	"&>>", "<<<", "<<-",
	"&&", "||", ";;", "|&", "<<", ">>", "&>", ">&", "<&", "<>", ">|",
	"<", ">", "(", ")", ";", "&", "|",
}

func isRedirectOperator(op string) bool {
	switch op {
	case "&>>", "<<<", "<<-", "<<", ">>", "&>", ">&", "<&", "<>", ">|", "<", ">":
		return true
	}
	return false
}

type heredocRequest struct {
	token     int
	delimiter string
	stripTabs bool
}

type shellLexer struct {
	src      []rune
	pos      int
	dialect  shellDialect
	tokens   []shellToken
	word     shellWord
	inWord   bool
	start    int
	pending  []heredocRequest
	awaiting *heredocRequest
}

// lexShell tokenizes command text. It fails on unterminated quotes and
// substitutions so text that cannot be understood is never treated as
// understood.
func lexShell(src string, dialect shellDialect) ([]shellToken, error) {
	l := &shellLexer{src: []rune(src), dialect: dialect}
	if err := l.run(); err != nil {
		return nil, err
	}
	return l.tokens, nil
}

func (l *shellLexer) run() error {
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '\n':
			l.flush()
			l.emitOperator(";")
			l.pos++
			l.readHeredocBodies()
		case c == ' ' || c == '\t' || c == '\r':
			l.flush()
			l.pos++
		case c == '#' && !l.inWord && l.dialect != dialectCmd:
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		case strings.ContainsRune(";&|<>()", c):
			if (c == '<' || c == '>') && l.wordIsFileDescriptor() {
				l.word, l.inWord = shellWord{}, false
			}
			l.flush()
			op := l.matchOperator()
			l.pos += len([]rune(op))
			l.emitOperator(op)
			if op == "<<" || op == "<<-" {
				l.awaiting = &heredocRequest{token: len(l.tokens) - 1, stripTabs: op == "<<-"}
			}
		default:
			if err := l.wordChar(); err != nil {
				return err
			}
		}
	}
	l.flush()
	return nil
}

func (l *shellLexer) matchOperator() string {
	rest := string(l.src[l.pos:])
	for _, op := range shellOperators {
		// Here-documents are POSIX-only; elsewhere << is two redirects.
		if l.dialect != dialectPOSIX && strings.HasPrefix(op, "<<") {
			continue
		}
		if strings.HasPrefix(rest, op) {
			return op
		}
	}
	return string(l.src[l.pos])
}

// wordIsFileDescriptor reports whether the word being built is a redirect's
// file-descriptor prefix, such as the 2 in 2>/dev/null.
func (l *shellLexer) wordIsFileDescriptor() bool {
	if !l.inWord || len(l.word.parts) != 1 || l.word.parts[0].kind != partText || l.word.parts[0].quoted {
		return false
	}
	text := l.word.parts[0].text
	if l.dialect == dialectPowerShell && text == "*" {
		return true
	}
	return text != "" && strings.Trim(text, "0123456789") == ""
}

func (l *shellLexer) emitOperator(op string) {
	l.tokens = append(l.tokens, shellToken{kind: tokenOperator, op: op})
}

func (l *shellLexer) begin() {
	if !l.inWord {
		l.inWord = true
		l.start = l.pos
	}
}

func (l *shellLexer) appendText(text string, quoted bool) {
	l.begin()
	if n := len(l.word.parts); n > 0 && l.word.parts[n-1].kind == partText && l.word.parts[n-1].quoted == quoted {
		l.word.parts[n-1].text += text
		return
	}
	l.word.parts = append(l.word.parts, wordPart{kind: partText, text: text, quoted: quoted})
}

func (l *shellLexer) appendPart(part wordPart) {
	l.begin()
	l.word.parts = append(l.word.parts, part)
}

func (l *shellLexer) flush() {
	if !l.inWord {
		return
	}
	l.word.raw = string(l.src[l.start:l.pos])
	l.tokens = append(l.tokens, shellToken{kind: tokenWord, word: l.word})
	if l.awaiting != nil {
		request := *l.awaiting
		request.delimiter = literalWordText(l.word)
		l.pending = append(l.pending, request)
		l.awaiting = nil
	}
	l.word, l.inWord = shellWord{}, false
}

// readHeredocBodies consumes the lines after a newline that belong to the
// here-documents opened on the line just ended.
func (l *shellLexer) readHeredocBodies() {
	for _, request := range l.pending {
		var body strings.Builder
		for l.pos < len(l.src) {
			end := l.pos
			for end < len(l.src) && l.src[end] != '\n' {
				end++
			}
			line := string(l.src[l.pos:end])
			l.pos = end
			if l.pos < len(l.src) {
				l.pos++
			}
			compare := line
			if request.stripTabs {
				compare = strings.TrimLeft(line, "\t")
			}
			if compare == request.delimiter {
				break
			}
			body.WriteString(line)
			body.WriteByte('\n')
		}
		l.tokens[request.token].body = body.String()
	}
	l.pending = nil
}

func (l *shellLexer) wordChar() error {
	switch l.dialect {
	case dialectPowerShell:
		return l.powerShellChar()
	case dialectCmd:
		return l.cmdChar()
	default:
		return l.posixChar()
	}
}

func (l *shellLexer) posixChar() error {
	c := l.src[l.pos]
	switch {
	case c == '\\':
		if l.pos+1 >= len(l.src) {
			return errors.New("dangling escape at end of command")
		}
		if l.src[l.pos+1] == '\n' {
			l.pos += 2
			return nil
		}
		l.appendText(string(l.src[l.pos+1]), true)
		l.pos += 2
	case c == '\'':
		end := indexRune(l.src, l.pos+1, '\'')
		if end < 0 {
			return errors.New("no closing single quote")
		}
		l.appendText(string(l.src[l.pos+1:end]), true)
		l.pos = end + 1
	case c == '$' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '\'':
		return l.ansiQuote()
	case c == '$' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '"':
		l.pos++
		return l.posixDoubleQuote()
	case c == '"':
		return l.posixDoubleQuote()
	case c == '$':
		return l.dollar(false)
	case c == '`':
		return l.backtick()
	case c == '~' && l.tildeAllowed():
		l.tilde()
	default:
		l.appendText(string(c), false)
		l.pos++
	}
	return nil
}

func (l *shellLexer) posixDoubleQuote() error {
	l.begin()
	l.pos++
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '"':
			l.pos++
			l.markQuotedEmpty()
			return nil
		case c == '\\' && l.pos+1 < len(l.src) && strings.ContainsRune("$`\"\\\n", l.src[l.pos+1]):
			if l.src[l.pos+1] != '\n' {
				l.appendText(string(l.src[l.pos+1]), true)
			}
			l.pos += 2
		case c == '$':
			if err := l.dollar(true); err != nil {
				return err
			}
		case c == '`':
			if err := l.backtick(); err != nil {
				return err
			}
		default:
			l.appendText(string(c), true)
			l.pos++
		}
	}
	return errors.New("no closing double quote")
}

// markQuotedEmpty keeps "" as a real, empty word rather than no word at all.
func (l *shellLexer) markQuotedEmpty() {
	if len(l.word.parts) == 0 {
		l.word.parts = append(l.word.parts, wordPart{kind: partText, quoted: true})
	}
}

func (l *shellLexer) ansiQuote() error {
	l.begin()
	l.pos += 2
	var text strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\'' {
			l.pos++
			l.appendText(text.String(), true)
			l.markQuotedEmpty()
			return nil
		}
		if c == '\\' && l.pos+1 < len(l.src) {
			next := l.src[l.pos+1]
			switch next {
			case 'n':
				text.WriteRune('\n')
			case 't':
				text.WriteRune('\t')
			case 'r':
				text.WriteRune('\r')
			default:
				text.WriteRune(next)
			}
			l.pos += 2
			continue
		}
		text.WriteRune(c)
		l.pos++
	}
	return errors.New("no closing ANSI-C quote")
}

// dollar handles every $-introduced expansion at the lexer position.
func (l *shellLexer) dollar(quoted bool) error {
	start := l.pos
	if l.pos+1 >= len(l.src) {
		l.appendText("$", quoted)
		l.pos++
		return nil
	}
	next := l.src[l.pos+1]
	switch {
	case next == '(' && l.pos+2 < len(l.src) && l.src[l.pos+2] == '(' && l.dialect == dialectPOSIX:
		end, err := matchClose(l.src, l.pos+3, '(', ')')
		if err != nil {
			return err
		}
		if end+1 >= len(l.src) || l.src[end+1] != ')' {
			return errors.New("no closing arithmetic expansion")
		}
		l.pos = end + 2
		l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos]), quoted: quoted})
	case next == '(':
		end, err := matchClose(l.src, l.pos+2, '(', ')')
		if err != nil {
			return err
		}
		if err := l.nest(string(l.src[l.pos+2 : end])); err != nil {
			return err
		}
		l.pos = end + 1
		l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos]), quoted: quoted})
	case next == '{':
		end := indexRune(l.src, l.pos+2, '}')
		if end < 0 {
			return errors.New("no closing brace in parameter expansion")
		}
		name := string(l.src[l.pos+2 : end])
		l.pos = end + 1
		if isShellName(name) {
			l.appendPart(wordPart{kind: partVariable, text: name, quoted: quoted})
		} else {
			l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos]), quoted: quoted})
		}
	case l.dialect == dialectPowerShell && hasFoldPrefix(l.src[l.pos+1:], "env:"):
		l.pos += 5
		name := l.readName()
		if name == "" {
			l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos]), quoted: quoted})
			return nil
		}
		l.appendPart(wordPart{kind: partVariable, text: name, quoted: quoted})
	case isNameStart(next):
		l.pos++
		name := l.readName()
		switch {
		case l.dialect == dialectPowerShell && strings.EqualFold(name, "HOME"):
			l.appendPart(wordPart{kind: partHome, text: "$" + name, quoted: quoted})
		case l.dialect == dialectPowerShell:
			l.appendPart(wordPart{kind: partDynamic, text: "$" + name, quoted: quoted})
		default:
			l.appendPart(wordPart{kind: partVariable, text: name, quoted: quoted})
		}
	case strings.ContainsRune("@*#?$!-0123456789", next):
		l.pos += 2
		l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos]), quoted: quoted})
	default:
		l.appendText("$", quoted)
		l.pos++
	}
	return nil
}

func (l *shellLexer) backtick() error {
	var inner strings.Builder
	start := l.pos
	l.pos++
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '`' {
			l.pos++
			if err := l.nest(inner.String()); err != nil {
				return err
			}
			l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos])})
			return nil
		}
		if c == '\\' && l.pos+1 < len(l.src) && strings.ContainsRune("`$\\", l.src[l.pos+1]) {
			inner.WriteRune(l.src[l.pos+1])
			l.pos += 2
			continue
		}
		inner.WriteRune(c)
		l.pos++
	}
	return errors.New("no closing backtick")
}

// nest lexes a command substitution so the walker can see the commands it
// runs; `echo $(rm -r x)` deletes as surely as `rm -r x`.
func (l *shellLexer) nest(inner string) error {
	tokens, err := lexShell(inner, l.dialect)
	if err != nil {
		return fmt.Errorf("command substitution: %w", err)
	}
	l.begin()
	l.word.nested = append(l.word.nested, tokens)
	return nil
}

// tildeAllowed reports whether a tilde here is a home-directory prefix: at the
// start of a word, or straight after the = of an assignment.
func (l *shellLexer) tildeAllowed() bool {
	if !l.inWord {
		return true
	}
	if len(l.word.parts) != 1 || l.word.parts[0].kind != partText || l.word.parts[0].quoted {
		return false
	}
	text := l.word.parts[0].text
	return strings.HasSuffix(text, "=") && isShellName(strings.TrimSuffix(text, "="))
}

func (l *shellLexer) tilde() {
	start := l.pos
	l.pos++
	for l.pos < len(l.src) && !strings.ContainsRune("/\\ \t\r\n;&|<>()", l.src[l.pos]) {
		l.pos++
	}
	if l.pos == start+1 {
		l.appendPart(wordPart{kind: partHome, text: "~"})
		return
	}
	// ~user, ~+ and ~- depend on state the hook cannot see.
	l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos])})
}

func (l *shellLexer) powerShellChar() error {
	c := l.src[l.pos]
	switch {
	case c == '`':
		if l.pos+1 >= len(l.src) {
			return errors.New("dangling escape at end of command")
		}
		if l.src[l.pos+1] == '\n' {
			l.pos += 2
			return nil
		}
		l.appendText(string(l.src[l.pos+1]), true)
		l.pos += 2
	case c == '\'':
		l.begin()
		l.pos++
		for l.pos < len(l.src) {
			if l.src[l.pos] == '\'' {
				if l.pos+1 < len(l.src) && l.src[l.pos+1] == '\'' {
					l.appendText("'", true)
					l.pos += 2
					continue
				}
				l.pos++
				l.markQuotedEmpty()
				return nil
			}
			l.appendText(string(l.src[l.pos]), true)
			l.pos++
		}
		return errors.New("no closing single quote")
	case c == '"':
		l.begin()
		l.pos++
		for l.pos < len(l.src) {
			switch ch := l.src[l.pos]; {
			case ch == '"' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '"':
				l.appendText(`"`, true)
				l.pos += 2
			case ch == '"':
				l.pos++
				l.markQuotedEmpty()
				return nil
			case ch == '`' && l.pos+1 < len(l.src):
				l.appendText(string(l.src[l.pos+1]), true)
				l.pos += 2
			case ch == '$':
				if err := l.dollar(true); err != nil {
					return err
				}
			default:
				l.appendText(string(ch), true)
				l.pos++
			}
		}
		return errors.New("no closing double quote")
	case c == '$':
		return l.dollar(false)
	case c == '~' && l.tildeAllowed():
		l.tilde()
	default:
		l.appendText(string(c), false)
		l.pos++
	}
	return nil
}

func (l *shellLexer) cmdChar() error {
	c := l.src[l.pos]
	switch {
	case c == '^':
		if l.pos+1 >= len(l.src) {
			return errors.New("dangling escape at end of command")
		}
		if l.src[l.pos+1] == '\n' {
			l.pos += 2
			return nil
		}
		l.appendText(string(l.src[l.pos+1]), true)
		l.pos += 2
	case c == '"':
		l.begin()
		l.pos++
		for l.pos < len(l.src) {
			switch ch := l.src[l.pos]; ch {
			case '"':
				l.pos++
				l.markQuotedEmpty()
				return nil
			case '%':
				l.percent(true)
			default:
				l.appendText(string(ch), true)
				l.pos++
			}
		}
		return errors.New("no closing double quote")
	case c == '%':
		l.percent(false)
	default:
		l.appendText(string(c), false)
		l.pos++
	}
	return nil
}

// percent handles cmd.exe %NAME% expansion, %% escapes, and batch parameters.
func (l *shellLexer) percent(quoted bool) {
	start := l.pos
	if l.pos+1 < len(l.src) && l.src[l.pos+1] == '%' {
		l.appendText("%", quoted)
		l.pos += 2
		return
	}
	if l.pos+1 < len(l.src) && (l.src[l.pos+1] == '~' || (l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9') || l.src[l.pos+1] == '*') {
		l.pos += 2
		for l.pos < len(l.src) && isNameChar(l.src[l.pos]) {
			l.pos++
		}
		l.appendPart(wordPart{kind: partDynamic, text: string(l.src[start:l.pos]), quoted: quoted})
		return
	}
	end := indexRune(l.src, l.pos+1, '%')
	if end > l.pos+1 && !strings.ContainsAny(string(l.src[l.pos+1:end]), " \t\n\"") {
		l.appendPart(wordPart{kind: partVariable, text: string(l.src[l.pos+1 : end]), quoted: quoted})
		l.pos = end + 1
		return
	}
	l.appendText("%", quoted)
	l.pos++
}

func (l *shellLexer) readName() string {
	start := l.pos
	for l.pos < len(l.src) && isNameChar(l.src[l.pos]) {
		l.pos++
	}
	return string(l.src[start:l.pos])
}

// matchClose finds the delimiter closing a group opened just before start,
// honouring nesting and quotes.
func matchClose(src []rune, start int, open, close rune) (int, error) {
	depth := 1
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
		case '\'':
			end := indexRune(src, i+1, '\'')
			if end < 0 {
				return 0, errors.New("no closing single quote")
			}
			i = end
		case '"':
			for i++; i < len(src) && src[i] != '"'; i++ {
				if src[i] == '\\' {
					i++
				}
			}
			if i >= len(src) {
				return 0, errors.New("no closing double quote")
			}
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("no closing %q", close)
}

func indexRune(src []rune, from int, want rune) int {
	for i := from; i < len(src); i++ {
		if src[i] == want {
			return i
		}
	}
	return -1
}

func hasFoldPrefix(src []rune, prefix string) bool {
	return len(src) >= len(prefix) && strings.EqualFold(string(src[:len(prefix)]), prefix)
}

func isNameStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isNameChar(r rune) bool { return isNameStart(r) || (r >= '0' && r <= '9') }

func isShellName(name string) bool {
	if name == "" || !isNameStart([]rune(name)[0]) {
		return false
	}
	for _, r := range name {
		if !isNameChar(r) {
			return false
		}
	}
	return true
}

// literalWordText concatenates a word's text as written, without expanding it.
// It names here-document delimiters and matches command names and flags.
func literalWordText(word shellWord) string {
	var text strings.Builder
	for _, part := range word.parts {
		switch part.kind {
		case partText:
			text.WriteString(part.text)
		case partVariable:
			text.WriteString("$" + part.text)
		default:
			text.WriteString(part.text)
		}
	}
	return text.String()
}
