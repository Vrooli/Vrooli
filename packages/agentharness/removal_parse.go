package agentharness

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// RemovalTarget is one thing a command would delete, resolved as far as the
// command text allows. A target the text cannot pin down carries the reason in
// Unresolved and no Path.
type RemovalTarget struct {
	Verb       string `json:"verb"`
	Raw        string `json:"raw"`
	Path       string `json:"path,omitempty"`
	Recursive  bool   `json:"recursive"`
	Glob       bool   `json:"glob,omitempty"`
	Unresolved string `json:"unresolved,omitempty"`
}

// maxShellNesting bounds how many interpreter layers (bash -c, eval, pwsh
// -Command, cmd /c) the walker re-parses before it stops guessing.
const maxShellNesting = 3

// removalEnv is what the walker needs to resolve words into paths.
type removalEnv struct {
	WorkingDirectory string
	Home             string
	Lookup           func(string) (string, bool)
}

type removalWalker struct {
	env     removalEnv
	dialect shellDialect
	vars    map[string]*string
	cwd     string
	depth   int
	targets *[]RemovalTarget
	invoked *bool
}

// parseRemovals lists what the command text would delete. It never executes
// anything; command text is data.
func parseRemovals(script string, dialect shellDialect, env removalEnv) []RemovalTarget {
	w := newRemovalWalker(env, dialect)
	w.script(script)
	return *w.targets
}

// parseArgvRemovals is parseRemovals for an argument vector that no shell
// will re-split or glob.
func parseArgvRemovals(argv []string, env removalEnv) []RemovalTarget {
	words := make([]shellWord, 0, len(argv))
	for _, arg := range argv {
		words = append(words, quotedWord(arg))
	}
	w := newRemovalWalker(env, dialectPOSIX)
	w.command(words, nil, false)
	return *w.targets
}

func newRemovalWalker(env removalEnv, dialect shellDialect) *removalWalker {
	targets := []RemovalTarget{}
	invoked := false
	return &removalWalker{env: env, dialect: dialect, vars: map[string]*string{}, cwd: filepath.Clean(env.WorkingDirectory), targets: &targets, invoked: &invoked}
}

func quotedWord(text string) shellWord {
	return shellWord{parts: []wordPart{{kind: partText, text: text, quoted: true}}, raw: text}
}

func (w *removalWalker) child() *removalWalker {
	vars := make(map[string]*string, len(w.vars))
	for name, value := range w.vars {
		vars[name] = value
	}
	return &removalWalker{env: w.env, dialect: w.dialect, vars: vars, cwd: w.cwd, depth: w.depth, targets: w.targets, invoked: w.invoked}
}

func (w *removalWalker) script(text string) {
	tokens, err := lexShell(text, w.dialect)
	if err != nil {
		if mentionsDeletion(text) {
			w.push(RemovalTarget{Verb: "unparsed", Raw: excerptText(text, 120), Unresolved: "the command text could not be parsed (" + err.Error() + ") and names a deletion command"})
		}
		return
	}
	w.tokens(tokens)
}

func (w *removalWalker) tokens(tokens []shellToken) {
	var words []shellWord
	var stdin []string
	piped := false
	var cwdStack []string
	flush := func(op string) {
		if len(words) > 0 {
			w.command(words, stdin, piped)
		}
		words, stdin = nil, nil
		piped = op == "|" || op == "|&"
	}
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.kind == tokenWord {
			words = append(words, token.word)
			continue
		}
		switch {
		case isRedirectOperator(token.op):
			if i+1 < len(tokens) && tokens[i+1].kind == tokenWord {
				target := tokens[i+1].word
				w.substitutions(target)
				if token.op == "<<<" {
					stdin = append(stdin, w.textOf(target))
				}
				i++
			}
			if token.op == "<<" || token.op == "<<-" {
				stdin = append(stdin, token.body)
			}
		case token.op == "(":
			flush(token.op)
			cwdStack = append(cwdStack, w.cwd)
		case token.op == ")":
			flush(token.op)
			if n := len(cwdStack); n > 0 {
				w.cwd, cwdStack = cwdStack[n-1], cwdStack[:n-1]
			}
		default:
			flush(token.op)
		}
	}
	flush("")
}

// substitutions walks the commands a word runs while it is being expanded.
// They run in a subshell, so their cd and assignments do not leak.
func (w *removalWalker) substitutions(word shellWord) {
	for _, nested := range word.nested {
		w.child().tokens(nested)
	}
}

func (w *removalWalker) command(words []shellWord, stdin []string, piped bool) {
	for _, word := range words {
		w.substitutions(word)
	}
	for len(words) > 0 && isLiteral(words[len(words)-1], "}") {
		words = words[:len(words)-1]
	}
	i := 0
	for i < len(words) && w.dialect == dialectPOSIX && isAssignmentWord(words[i]) {
		i++
	}
	if i == len(words) {
		for _, word := range words {
			w.assign(word)
		}
		return
	}
	for i < len(words) {
		name := w.commandName(words[i])
		if name == "" {
			return
		}
		if w.isKeyword(name) {
			i++
			continue
		}
		if w.opensCompound(name) {
			return
		}
		next, wrapped := w.skipWrapper(name, words, i)
		if !wrapped {
			break
		}
		i = next
	}
	if i >= len(words) {
		return
	}
	w.dispatch(w.commandName(words[i]), words[i+1:], stdin, piped)
}

func (w *removalWalker) isKeyword(name string) bool {
	switch name {
	case "{", "}", "!", "then", "else", "do", "done", "fi", "if", "elif", "while", "until", "time", "try", "finally", "elseif":
		return true
	}
	return false
}

// opensCompound reports the keywords whose words are not a command: the head
// of a for loop, a case subject, a function definition.
func (w *removalWalker) opensCompound(name string) bool {
	switch name {
	case "for", "foreach", "case", "select", "function", "esac", "in", "catch":
		return true
	}
	return false
}

func (w *removalWalker) skipWrapper(name string, words []shellWord, i int) (int, bool) {
	option := func(j int) string { return literalWordText(words[j]) }
	j := i + 1
	switch name {
	case "sudo", "doas":
		for j < len(words) {
			t := option(j)
			if t == "--" {
				return j + 1, true
			}
			if !strings.HasPrefix(t, "-") {
				break
			}
			switch t {
			case "-u", "-g", "-h", "-p", "-C", "-D", "-r", "-t", "-U", "-T", "--user", "--group", "--host", "--prompt", "--chdir":
				j++
			}
			j++
		}
		return j, true
	case "nice", "ionice":
		for j < len(words) && strings.HasPrefix(option(j), "-") {
			switch option(j) {
			case "-n", "-c", "-p", "-P", "-u", "--adjustment", "--class", "--classdata":
				j++
			}
			j++
		}
		return j, true
	case "nohup", "builtin", "busybox", "chronic", "unbuffer", "call":
		return j, true
	case "exec":
		for j < len(words) && strings.HasPrefix(option(j), "-") {
			if option(j) == "-a" {
				j++
			}
			j++
		}
		return j, true
	case "command":
		for j < len(words) && strings.HasPrefix(option(j), "-") {
			if option(j) == "-v" || option(j) == "-V" {
				// A lookup, not an invocation: nothing runs.
				return len(words), true
			}
			j++
		}
		return j, true
	case "env":
		for j < len(words) {
			t := option(j)
			switch {
			case t == "--":
				return j + 1, true
			case t == "-S" || t == "--split-string":
				if j+1 < len(words) {
					w.nestedScript(w.textOf(words[j+1]), dialectPOSIX, "env -S")
				}
				return len(words), true
			case t == "-u" || t == "-C" || t == "--unset" || t == "--chdir":
				j += 2
			case strings.HasPrefix(t, "-") || isAssignmentWord(words[j]):
				j++
			default:
				return j, true
			}
		}
		return j, true
	case "timeout":
		for j < len(words) && strings.HasPrefix(option(j), "-") {
			switch option(j) {
			case "-s", "-k", "--signal", "--kill-after":
				j++
			}
			j++
		}
		return j + 1, true
	case "stdbuf":
		for j < len(words) && strings.HasPrefix(option(j), "-") {
			switch option(j) {
			case "-i", "-o", "-e":
				j++
			}
			j++
		}
		return j, true
	}
	return i, false
}

func (w *removalWalker) dispatch(name string, args []shellWord, stdin []string, piped bool) {
	switch w.dialect {
	case dialectPowerShell:
		switch name {
		case "remove-item", "ri", "rm", "rmdir", "del", "erase", "rd":
			w.removeItem("Remove-Item", args, piped, true)
			return
		case "clear-content", "clc":
			w.removeItem("Clear-Content", args, piped, false)
			return
		case "set-location", "sl", "cd", "chdir", "push-location", "pushd":
			w.cwd = w.powerShellLocation(args)
			return
		case "pop-location", "popd":
			w.cwd = ""
			return
		case "invoke-expression", "iex":
			w.nestedScript(w.joined(args), dialectPowerShell, "Invoke-Expression")
			return
		}
	case dialectCmd:
		switch name {
		case "del", "erase":
			// del on a directory deletes the files inside it.
			w.cmdRemove(name, args, true)
			return
		case "rd", "rmdir":
			w.cmdRemove(name, args, false)
			return
		case "cd", "chdir", "pushd":
			w.cwd = w.cmdLocation(args)
			return
		case "popd":
			w.cwd = ""
			return
		case "set":
			// cmd expands %NAME% for a whole line before running it, so a value
			// set on this line is not what a later %NAME% on it reads.
			if text := w.joined(args); strings.Contains(text, "=") {
				w.vars[strings.TrimSpace(strings.SplitN(text, "=", 2)[0])] = nil
			}
			return
		}
	}
	switch name {
	case "rm":
		w.rm(args)
	case "rmdir", "unlink":
		w.plainTargets(name, args, nil)
	case "shred":
		w.plainTargets(name, args, map[string]bool{"-n": true, "-s": true, "--iterations": true, "--size": true, "--random-source": true})
	case "truncate":
		w.plainTargets(name, args, map[string]bool{"-s": true, "--size": true, "-r": true, "--reference": true})
	case "find":
		w.find(args)
	case "rsync":
		w.rsync(args)
	case "xargs":
		for k := range args {
			if w.probe(args[k:]) {
				*w.invoked = true
				w.push(RemovalTarget{Verb: "xargs", Raw: excerptText(w.joined(args), 120), Unresolved: "xargs feeds a deletion command targets from standard input, which the hook cannot see"})
				return
			}
		}
	case "bash", "sh", "zsh", "dash", "ksh", "ash", "mksh", "yash":
		w.shellInterpreter(name, args, stdin)
	case "eval":
		w.nestedScript(w.joined(args), dialectPOSIX, "eval")
	case "pwsh", "powershell":
		w.powerShellHost(name, args)
	case "cmd":
		w.cmdHost(args)
	case "cd", "pushd", "chdir":
		w.cwd = w.posixLocation(args)
	case "popd":
		w.cwd = ""
	case "remove-item":
		w.removeItem("Remove-Item", args, piped, true)
	case "clear-content":
		w.removeItem("Clear-Content", args, piped, false)
	case "export", "declare", "typeset", "local", "readonly":
		for _, arg := range args {
			if isAssignmentWord(arg) {
				w.assign(arg)
			}
		}
	}
}

func (w *removalWalker) rm(args []shellWord) {
	*w.invoked = true
	recursive, endOptions := false, false
	var operands []shellWord
	for _, arg := range args {
		t := literalWordText(arg)
		switch {
		case !endOptions && t == "--":
			endOptions = true
		case !endOptions && strings.HasPrefix(t, "--"):
			if t == "--recursive" {
				recursive = true
			}
		case !endOptions && len(t) > 1 && t[0] == '-':
			if strings.ContainsAny(t[1:], "rR") {
				recursive = true
			}
		default:
			operands = append(operands, arg)
		}
	}
	for _, operand := range operands {
		w.target("rm", operand, recursive)
	}
}

func (w *removalWalker) plainTargets(verb string, args []shellWord, argOptions map[string]bool) {
	*w.invoked = true
	endOptions := false
	for j := 0; j < len(args); j++ {
		t := literalWordText(args[j])
		switch {
		case !endOptions && t == "--":
			endOptions = true
		case !endOptions && argOptions[t]:
			j++
		case !endOptions && len(t) > 1 && t[0] == '-':
		default:
			w.target(verb, args[j], false)
		}
	}
}

// find deletes only with -delete or an -exec that runs a deletion command, and
// then everything under each starting point is in reach.
func (w *removalWalker) find(args []shellWord) {
	j := 0
	for j < len(args) {
		t := literalWordText(args[j])
		if t == "-H" || t == "-L" || t == "-P" || strings.HasPrefix(t, "-O") {
			j++
			continue
		}
		if t == "-D" {
			j += 2
			continue
		}
		break
	}
	var roots []shellWord
	for ; j < len(args); j++ {
		t := literalWordText(args[j])
		if strings.HasPrefix(t, "-") || t == "(" || t == "!" || t == ")" || t == "," {
			break
		}
		roots = append(roots, args[j])
	}
	deletes := false
	for k := j; k < len(args); k++ {
		switch literalWordText(args[k]) {
		case "-delete":
			deletes = true
		case "-exec", "-execdir", "-ok", "-okdir":
			end := k + 1
			for end < len(args) && literalWordText(args[end]) != ";" && literalWordText(args[end]) != "+" {
				end++
			}
			if w.probe(args[k+1 : end]) {
				deletes = true
			}
			k = end
		}
	}
	if !deletes {
		return
	}
	*w.invoked = true
	if len(roots) == 0 {
		roots = []shellWord{quotedWord(".")}
	}
	for _, root := range roots {
		w.target("find", root, true)
	}
}

// rsync --delete removes whatever in the destination the source lacks.
func (w *removalWalker) rsync(args []shellWord) {
	deletes := false
	var operands []shellWord
	for _, arg := range args {
		t := literalWordText(arg)
		if strings.HasPrefix(t, "--delete") {
			deletes = true
		}
		if !strings.HasPrefix(t, "-") {
			operands = append(operands, arg)
		}
	}
	if !deletes || len(operands) < 2 {
		return
	}
	destination := operands[len(operands)-1]
	if text, _, _ := w.expand(destination); strings.Contains(text, ":") && !filepath.IsAbs(text) {
		return
	}
	*w.invoked = true
	w.target("rsync --delete", destination, true)
}

// probe reports whether the words, run as a command, would invoke a deletion
// command, without recording anything.
func (w *removalWalker) probe(words []shellWord) bool {
	if len(words) == 0 {
		return false
	}
	targets := []RemovalTarget{}
	invoked := false
	child := w.child()
	child.targets, child.invoked = &targets, &invoked
	child.command(words, nil, false)
	return invoked || len(targets) > 0
}

func (w *removalWalker) shellInterpreter(name string, args []shellWord, stdin []string) {
	for j := 0; j < len(args); j++ {
		t := literalWordText(args[j])
		switch {
		case strings.HasPrefix(t, "--"):
		case t == "-o" || t == "+o" || t == "-O" || t == "+O":
			j++
		case strings.HasPrefix(t, "-") && strings.ContainsRune(t[1:], 'c'):
			if j+1 < len(args) {
				w.nestedScript(w.textOf(args[j+1]), dialectPOSIX, name+" -c")
			}
			return
		case strings.HasPrefix(t, "-") || strings.HasPrefix(t, "+"):
		default:
			// A script file: its contents are not in the command text.
			return
		}
	}
	for _, body := range stdin {
		w.nestedScript(body, dialectPOSIX, name)
	}
}

func (w *removalWalker) powerShellHost(name string, args []shellWord) {
	saved := w.cwd
	defer func() { w.cwd = saved }()
	for j := 0; j < len(args); j++ {
		t := strings.ToLower(literalWordText(args[j]))
		if !strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "/") {
			if name == "powershell" {
				w.nestedScript(w.joined(args[j:]), dialectPowerShell, name)
			}
			return
		}
		param := strings.TrimLeft(t, "-/")
		switch {
		case param == "c" || paramPrefix(param, "command", 3):
			w.nestedScript(w.joined(args[j+1:]), dialectPowerShell, name+" -Command")
			return
		case param == "ec" || param == "e" || paramPrefix(param, "encodedcommand", 2):
			*w.invoked = true
			w.push(RemovalTarget{Verb: name, Raw: excerptText(w.joined(args), 120), Unresolved: "an encoded PowerShell command cannot be inspected"})
			return
		case param == "f" || paramPrefix(param, "file", 2):
			return
		case param == "wd" || paramPrefix(param, "workingdirectory", 2):
			if j+1 < len(args) {
				w.cwd = w.resolveDirectory(args[j+1])
			}
			j++
		case param == "ep" || param == "ex" || param == "of" || param == "if" || param == "w" || param == "v" ||
			paramPrefix(param, "executionpolicy", 3) || paramPrefix(param, "outputformat", 3) || paramPrefix(param, "inputformat", 3) ||
			paramPrefix(param, "configurationname", 3) || paramPrefix(param, "settingsfile", 3) || paramPrefix(param, "windowstyle", 3) || paramPrefix(param, "custompipename", 3):
			j++
		}
	}
}

func (w *removalWalker) cmdHost(args []shellWord) {
	for j := 0; j < len(args); j++ {
		t := strings.ToLower(literalWordText(args[j]))
		switch {
		case t == "/c" || t == "/k" || t == "//c" || t == "//k":
			w.nestedScript(w.joined(args[j+1:]), dialectCmd, "cmd /c")
			return
		case strings.HasPrefix(t, "/"):
		default:
			return
		}
	}
}

// removeItem reads PowerShell's Remove-Item and Clear-Content parameters.
func (w *removalWalker) removeItem(verb string, args []shellWord, piped, removes bool) {
	recursive, whatIf := false, false
	var paths []shellWord
	for j := 0; j < len(args); j++ {
		t := literalWordText(args[j])
		quotedFirst := len(args[j].parts) > 0 && args[j].parts[0].quoted
		if !strings.HasPrefix(t, "-") || len(t) < 2 || quotedFirst {
			paths = append(paths, args[j])
			continue
		}
		name, value, hasValue := strings.Cut(strings.ToLower(t[1:]), ":")
		switch {
		case paramPrefix(name, "recurse", 1):
			recursive = removes && (!hasValue || value != "$false")
		case paramPrefix(name, "whatif", 2):
			whatIf = !hasValue || value != "$false"
		case paramPrefix(name, "path", 1) || paramPrefix(name, "literalpath", 1) || name == "lp" || name == "pspath":
			if hasValue {
				paths = append(paths, quotedWord(t[strings.IndexByte(t, ':')+1:]))
			} else if j+1 < len(args) {
				paths = append(paths, args[j+1])
				j++
			}
		case !hasValue && powerShellValueParameter(name):
			j++
		}
	}
	if whatIf {
		return
	}
	*w.invoked = true
	if len(paths) == 0 {
		if piped {
			w.push(RemovalTarget{Verb: verb, Raw: verb, Unresolved: verb + " receives its targets from the pipeline, which the hook cannot see"})
		}
		return
	}
	for _, path := range paths {
		text, glob, dynamic := w.expand(path)
		for _, element := range strings.Split(text, ",") {
			element = strings.TrimSpace(element)
			// The FileSystem provider expands a leading ~ in every path, not
			// only where the parser sees a word start.
			if element == "~" || strings.HasPrefix(element, "~/") || strings.HasPrefix(element, `~\`) {
				if strings.TrimSpace(w.env.Home) == "" {
					dynamic = "the home directory is unknown"
				}
				element = w.env.Home + element[1:]
			}
			if element != "" || dynamic != "" {
				w.addTarget(verb, path.raw, element, glob, dynamic, recursive)
			}
		}
	}
}

func powerShellValueParameter(name string) bool {
	switch name {
	case "ea", "wa", "infa", "ev", "wv", "iv", "ov", "ob", "pv", "proga":
		return true
	}
	for _, full := range []string{"filter", "include", "exclude", "credential", "stream", "erroraction", "warningaction", "informationaction", "errorvariable", "warningvariable", "informationvariable", "outvariable", "outbuffer", "pipelinevariable", "progressaction"} {
		if paramPrefix(name, full, 3) {
			return true
		}
	}
	return false
}

// paramPrefix reports whether name abbreviates the full parameter, as
// PowerShell accepts any unambiguous prefix.
func paramPrefix(name, full string, minimum int) bool {
	return len(name) >= minimum && strings.HasPrefix(full, name)
}

func (w *removalWalker) cmdRemove(verb string, args []shellWord, recursive bool) {
	*w.invoked = true
	var operands []shellWord
	for _, arg := range args {
		t := strings.ToLower(literalWordText(arg))
		if strings.HasPrefix(t, "/") && (len(t) == 2 || t[2] == ':') {
			if t == "/s" {
				recursive = true
			}
			continue
		}
		operands = append(operands, arg)
	}
	for _, operand := range operands {
		w.target(verb, operand, recursive)
	}
}

func (w *removalWalker) posixLocation(args []shellWord) string {
	for _, arg := range args {
		switch t := literalWordText(arg); t {
		case "-P", "-L", "-e", "-@":
			continue
		case "-":
			return ""
		default:
			return w.resolveDirectory(arg)
		}
	}
	return w.env.Home
}

func (w *removalWalker) powerShellLocation(args []shellWord) string {
	for j := 0; j < len(args); j++ {
		t := strings.ToLower(literalWordText(args[j]))
		if strings.HasPrefix(t, "-") {
			if (paramPrefix(t[1:], "path", 1) || paramPrefix(t[1:], "literalpath", 1)) && j+1 < len(args) {
				return w.resolveDirectory(args[j+1])
			}
			continue
		}
		return w.resolveDirectory(args[j])
	}
	return w.env.Home
}

func (w *removalWalker) cmdLocation(args []shellWord) string {
	for _, arg := range args {
		if strings.EqualFold(literalWordText(arg), "/d") {
			continue
		}
		return w.resolveDirectory(arg)
	}
	return w.cwd
}

func (w *removalWalker) resolveDirectory(word shellWord) string {
	text, glob, dynamic := w.expand(word)
	if dynamic != "" || glob {
		return ""
	}
	path, reason := w.absolute(text)
	if reason != "" {
		return ""
	}
	return path
}

func (w *removalWalker) nestedScript(text string, dialect shellDialect, via string) {
	if w.depth >= maxShellNesting {
		if mentionsDeletion(text) {
			*w.invoked = true
			w.push(RemovalTarget{Verb: via, Raw: excerptText(text, 120), Unresolved: fmt.Sprintf("a deletion nested more than %d interpreters deep cannot be inspected", maxShellNesting)})
		}
		return
	}
	child := w.child()
	child.depth++
	child.dialect = dialect
	child.script(text)
}

func (w *removalWalker) assign(word shellWord) {
	first := word.parts[0]
	name, rest, _ := strings.Cut(first.text, "=")
	value := shellWord{parts: append([]wordPart{{kind: partText, text: rest, quoted: first.quoted}}, word.parts[1:]...)}
	text, _, dynamic := w.expand(value)
	if dynamic != "" {
		w.vars[name] = nil
		return
	}
	w.vars[name] = &text
}

func (w *removalWalker) target(verb string, word shellWord, recursive bool) {
	text, glob, dynamic := w.expand(word)
	w.addTarget(verb, word.raw, text, glob, dynamic, recursive)
}

func (w *removalWalker) addTarget(verb, raw, text string, glob bool, dynamic string, recursive bool) {
	*w.invoked = true
	target := RemovalTarget{Verb: verb, Raw: raw, Recursive: recursive, Glob: glob}
	if target.Raw == "" {
		target.Raw = text
	}
	if dynamic != "" {
		target.Unresolved = dynamic
		w.push(target)
		return
	}
	if text == "" {
		return
	}
	if glob {
		// A pattern reaches anything its static directory holds; the target
		// becomes that directory's contents.
		text = globBase(text, w.globChars())
	}
	path, reason := w.absolute(text)
	if reason != "" {
		target.Unresolved = reason
	} else {
		target.Path = path
	}
	w.push(target)
}

func (w *removalWalker) push(target RemovalTarget) { *w.targets = append(*w.targets, target) }

func (w *removalWalker) absolute(text string) (string, string) {
	if runtime.GOOS == "windows" {
		if w.dialect == dialectPOSIX && len(text) >= 2 && text[0] == '/' && isNameStart(rune(text[1])) && (len(text) == 2 || text[2] == '/') {
			// Git Bash spells C:\x as /c/x.
			text = strings.ToUpper(text[1:2]) + ":\\" + strings.TrimPrefix(text[2:], "/")
		}
		if volume := filepath.VolumeName(text); volume != "" && !filepath.IsAbs(text) {
			return "", "a drive-relative path depends on that drive's current directory"
		}
		if !filepath.IsAbs(text) && (strings.HasPrefix(text, `\`) || strings.HasPrefix(text, "/")) {
			if w.cwd == "" || w.cwd == "." {
				return "", "a rooted path without a drive depends on the current drive"
			}
			return filepath.Clean(filepath.VolumeName(w.cwd) + text), ""
		}
	}
	if filepath.IsAbs(text) {
		return filepath.Clean(text), ""
	}
	if w.cwd == "" || w.cwd == "." {
		return "", "the path is relative and the working directory is unknown"
	}
	return filepath.Join(w.cwd, text), ""
}

func (w *removalWalker) globChars() string {
	if w.dialect == dialectPOSIX {
		return "*?[{"
	}
	return "*?["
}

// globBase returns the directory a pattern cannot escape: everything before
// the last separator ahead of the first wildcard.
func globBase(text, wildcards string) string {
	index := strings.IndexAny(text, wildcards)
	if index < 0 {
		return text
	}
	prefix := text[:index]
	separator := strings.LastIndexAny(prefix, `/\`)
	if separator < 0 {
		return "."
	}
	return prefix[:separator+1]
}

// expand resolves a word the way the shell would at this point in the
// command. dynamic explains the first part that only exists at run time.
func (w *removalWalker) expand(word shellWord) (string, bool, string) {
	var text strings.Builder
	glob := false
	dynamic := ""
	for _, part := range word.parts {
		switch part.kind {
		case partText:
			text.WriteString(part.text)
			if !part.quoted && w.hasGlob(part.text) {
				glob = true
			}
		case partVariable:
			value, ok := w.lookup(part.text)
			if !ok {
				if dynamic == "" {
					dynamic = w.unsetReason(part.text)
				}
				text.WriteString("$" + part.text)
				continue
			}
			text.WriteString(value)
			if !part.quoted && w.dialect == dialectPOSIX && w.hasGlob(value) {
				glob = true
			}
		case partHome:
			if strings.TrimSpace(w.env.Home) == "" {
				if dynamic == "" {
					dynamic = "the home directory is unknown"
				}
				text.WriteString(part.text)
				continue
			}
			text.WriteString(w.env.Home)
		case partDynamic:
			if dynamic == "" {
				dynamic = part.text + " is only known when the command runs"
			}
			text.WriteString(part.text)
		}
	}
	return text.String(), glob, dynamic
}

// textOf expands a word that is itself command text, keeping run-time parts
// as written so the nested parse sees them.
func (w *removalWalker) textOf(word shellWord) string {
	text, _, _ := w.expand(word)
	return text
}

func (w *removalWalker) joined(words []shellWord) string {
	texts := make([]string, 0, len(words))
	for _, word := range words {
		texts = append(texts, w.textOf(word))
	}
	return strings.Join(texts, " ")
}

func (w *removalWalker) hasGlob(text string) bool {
	if strings.ContainsAny(text, "*?[") {
		return true
	}
	if w.dialect != dialectPOSIX {
		return false
	}
	open := strings.IndexByte(text, '{')
	if open < 0 {
		return false
	}
	close := strings.IndexByte(text[open:], '}')
	return close > 0 && (strings.Contains(text[open:open+close], ",") || strings.Contains(text[open:open+close], ".."))
}

func (w *removalWalker) lookup(name string) (string, bool) {
	if value, ok := w.vars[name]; ok {
		if value == nil {
			return "", false
		}
		return *value, true
	}
	if w.env.Lookup == nil {
		return "", false
	}
	return w.env.Lookup(name)
}

func (w *removalWalker) unsetReason(name string) string {
	if value, ok := w.vars[name]; ok && value == nil {
		return "$" + name + " is assigned when the command runs"
	}
	return "$" + name + " is not set in the hook's environment"
}

func (w *removalWalker) commandName(word shellWord) string {
	text, _, dynamic := w.expand(word)
	if dynamic != "" {
		return ""
	}
	if index := strings.LastIndexAny(text, `/\`); index >= 0 && index < len(text)-1 {
		text = text[index+1:]
	}
	return strings.TrimSuffix(strings.ToLower(text), ".exe")
}

func isAssignmentWord(word shellWord) bool {
	if len(word.parts) == 0 || word.parts[0].kind != partText || word.parts[0].quoted {
		return false
	}
	name, _, found := strings.Cut(word.parts[0].text, "=")
	return found && isShellName(name)
}

func isLiteral(word shellWord, text string) bool {
	return len(word.parts) == 1 && word.parts[0].kind == partText && !word.parts[0].quoted && word.parts[0].text == text
}

// deletionWords are the commands that delete by themselves. find and xargs
// are absent: they delete only through -delete or a deletion command, which
// this list or the -delete check already catches.
var deletionWords = map[string]bool{
	"rm": true, "rmdir": true, "unlink": true, "shred": true, "truncate": true,
	"remove-item": true, "ri": true, "del": true, "erase": true, "rd": true, "clear-content": true, "clc": true,
}

// mentionsDeletion decides whether text the walker cannot parse still names a
// deletion, in which case it must not pass as harmless.
func mentionsDeletion(text string) bool {
	for _, field := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(isNameChar(r) || r == '-')
	}) {
		if deletionWords[strings.TrimSuffix(field, ".exe")] || field == "-delete" {
			return true
		}
	}
	return false
}

func excerptText(text string, limit int) string {
	flattened := strings.Join(strings.Fields(text), " ")
	if runes := []rune(flattened); len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return flattened
}
