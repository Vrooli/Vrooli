package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The PreToolUse guard is the native matcher paired with every Bash deny rule.
// It is pure Go on purpose: the previous implementation was a bash script that
// shelled out to python3, which made the deny hook unavailable on any host
// without both. Command text is data here — the guard parses and inspects it
// and never executes it.

// Claude Code's hook contract uses the process exit code as the decision:
// zero lets the tool call continue, two denies it.
const (
	GuardExitContinue = 0
	GuardExitDeny     = 2
)

// Operator-tunable log limits, matching the environment variable names the
// hook has always honoured.
const (
	defaultGuardLogMaxBytes    = 8 << 20
	defaultGuardLogCommandMax  = 300
	guardLogMaxBytesEnv        = "VROOLI_HOOK_LOG_MAX_BYTES"
	guardLogCommandMaxCharsEnv = "VROOLI_HOOK_LOG_CMD_MAX_CHARS"
)

// shellPunctuation are the characters tokenized as standalone operators rather
// than word characters, mirroring POSIX shell lexing.
const shellPunctuation = ";&|<>()"

// shellControlOperators are the operators whose presence means the command is
// compound, so single-command target extraction cannot be trusted.
var shellControlOperators = map[string]struct{}{
	";": {}, "&&": {}, "||": {}, "|": {}, "<": {}, ">": {}, "(": {}, ")": {},
}

var (
	destructiveCommandPattern = regexp.MustCompile(`(^|\s)(sudo\s+)?(rm|find|truncate)(\s|$)`)
	shellInterpreterPattern   = regexp.MustCompile(`(^|\s)(bash|sh|zsh)\s+-c(\s|$)`)
	unresolvedVariablePattern = regexp.MustCompile(`\$\{?[A-Za-z_][A-Za-z0-9_]*\}?`)
	expandableVariable        = regexp.MustCompile(`\$\{[A-Za-z_][A-Za-z0-9_]*\}|\$[A-Za-z_][A-Za-z0-9_]*`)
)

// GuardEnv is the resolved host context a single guard decision reads. It is a
// value rather than ambient process state so the decision is testable in
// process and behaves identically when invoked as a hook.
type GuardEnv struct {
	// Home is the protected user home directory.
	Home string
	// RepoRoot is the protected source checkout, when one is known.
	RepoRoot string
	// EphemeralRoots are the only directories a destructive command may target.
	EphemeralRoots []string
	// Lookup resolves environment variables found in a destructive path.
	Lookup func(string) (string, bool)
	// LogPath is the append-only audit log for hook decisions.
	LogPath string
	// LogMaxBytes is the size at which the audit log rotates.
	LogMaxBytes int64
	// LogCommandMax caps how much command text one log line carries.
	LogCommandMax int
}

// LoadGuardEnv resolves the guard context from the process environment.
func LoadGuardEnv() GuardEnv {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	roots := []string{"/tmp", "/var/tmp"}
	if temporary := strings.TrimSpace(os.TempDir()); temporary != "" {
		roots = append(roots, temporary)
	}
	for _, extra := range filepath.SplitList(os.Getenv("VROOLI_EPHEMERAL_ROOTS")) {
		if strings.TrimSpace(extra) != "" {
			roots = append(roots, extra)
		}
	}
	return GuardEnv{
		Home:           home,
		RepoRoot:       strings.TrimSpace(os.Getenv("VROOLI_ROOT")),
		EphemeralRoots: roots,
		Lookup:         os.LookupEnv,
		LogPath:        filepath.Join(home, ".claude", HookStateDirName, "log"),
		LogMaxBytes:    positiveEnvInt(guardLogMaxBytesEnv, defaultGuardLogMaxBytes),
		LogCommandMax:  int(positiveEnvInt(guardLogCommandMaxCharsEnv, defaultGuardLogCommandMax)),
	}
}

func positiveEnvInt(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	var value int64
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (e GuardEnv) lookup(name string) (string, bool) {
	if e.Lookup == nil {
		return "", false
	}
	return e.Lookup(name)
}

// GuardDecision is the outcome of one PreToolUse evaluation.
type GuardDecision struct {
	Exit   int
	Reason string
}

// Denied reports whether the decision stops the tool call.
func (d GuardDecision) Denied() bool { return d.Exit != GuardExitContinue }

func allowGuard() GuardDecision { return GuardDecision{Exit: GuardExitContinue} }

func denyGuard(reason string) GuardDecision {
	return GuardDecision{Exit: GuardExitDeny, Reason: reason}
}

// ExtractHookCommand reads the Bash command text out of a PreToolUse event.
// The payload is treated strictly as data.
func ExtractHookCommand(data []byte) (string, error) {
	malformed := errors.New("malformed hook input or missing tool_input.command")
	var event struct {
		ToolInput struct {
			Command *string `json:"command"`
		} `json:"tool_input"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return "", malformed
	}
	if event.ToolInput.Command == nil || *event.ToolInput.Command == "" {
		return "", malformed
	}
	return *event.ToolInput.Command, nil
}

// EvaluateGuard decides one command. Filesystem deletion is checked by
// resolved paths first, then the remaining native deny patterns are matched.
func EvaluateGuard(command string, patterns []string, env GuardEnv) GuardDecision {
	if decision := evaluateDestructivePaths(command, env); decision.Denied() {
		return decision
	}
	return evaluateDenyPatterns(command, patterns, env)
}

func evaluateDestructivePaths(command string, env GuardEnv) GuardDecision {
	tokens, err := splitShellTokens(command)
	if err != nil {
		return denyGuard("malformed shell command")
	}
	destructive := destructiveCommandPattern.MatchString(command)
	if containsControlOperator(tokens) {
		if destructive {
			return denyGuard("compound destructive shell command requires review")
		}
		return allowGuard()
	}
	if destructive && shellInterpreterPattern.MatchString(command) {
		return denyGuard("destructive command through a shell interpreter requires review")
	}
	if index := tokenIndex(tokens, "rm"); index >= 0 {
		return checkDestructiveTargets(removeTargets(tokens[index+1:]), "rm has no explicit target", true, env)
	}
	if index := tokenIndex(tokens, "find"); index >= 0 && containsToken(tokens[index+1:], "-delete") {
		return checkDestructiveTargets(leadingOperands(tokens[index+1:]), "find -delete has no explicit root", false, env)
	}
	if index := tokenIndex(tokens, "truncate"); index >= 0 {
		return checkDestructiveTargets(truncateTargets(tokens[index+1:]), "truncate has no explicit target", false, env)
	}
	return allowGuard()
}

func checkDestructiveTargets(targets []string, emptyReason string, rejectGlobs bool, env GuardEnv) GuardDecision {
	if len(targets) == 0 {
		return denyGuard(emptyReason)
	}
	for _, target := range targets {
		if rejectGlobs && strings.ContainsAny(target, "*?[") {
			return denyGuard("destructive path globs require review")
		}
		if decision := checkDestructiveTarget(target, env); decision.Denied() {
			return decision
		}
	}
	return allowGuard()
}

func checkDestructiveTarget(raw string, env GuardEnv) GuardDecision {
	value := expandUserHome(expandVariables(raw, env), env.Home)
	if unresolvedVariablePattern.MatchString(value) {
		return denyGuard("unresolved environment variable in destructive path")
	}
	if !filepath.IsAbs(value) {
		return denyGuard("destructive path is not absolute")
	}
	path := realPath(value)
	if path == "/" || path == string(filepath.Separator) {
		return denyGuard("destructive target is a protected root")
	}
	for _, protected := range []string{env.Home, env.RepoRoot} {
		resolved := realPath(protected)
		if resolved == "" {
			continue
		}
		if path == resolved || isWithin(path, resolved) {
			return denyGuard("destructive target is a protected root")
		}
	}
	if isDepthOneSystemDirectory(path) {
		return denyGuard("destructive target is a depth-one system directory")
	}
	for _, root := range env.EphemeralRoots {
		resolved := realPath(root)
		if resolved == "" {
			continue
		}
		if path != resolved && isWithin(path, resolved) {
			return allowGuard()
		}
	}
	return denyGuard("destructive target is outside an approved ephemeral root")
}

// isDepthOneSystemDirectory reports whether the path is a direct child of the
// filesystem root, such as /etc, which is never an acceptable target.
func isDepthOneSystemDirectory(path string) bool {
	return strings.HasPrefix(path, "/") && strings.Count(path, "/") == 1
}

func isWithin(path, root string) bool {
	return strings.HasPrefix(path, strings.TrimSuffix(root, string(filepath.Separator))+string(filepath.Separator))
}

// realPath resolves symlinks without requiring the path to exist, so a target
// that has not been created yet still resolves through its existing parents.
func realPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(realPath(parent), filepath.Base(path))
}

func expandVariables(value string, env GuardEnv) string {
	return expandableVariable.ReplaceAllStringFunc(value, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(match, "$"), "{"), "}")
		if resolved, ok := env.lookup(name); ok {
			return resolved
		}
		return match
	})
}

func expandUserHome(value, home string) string {
	if home == "" {
		return value
	}
	if value == "~" {
		return home
	}
	if strings.HasPrefix(value, "~/") {
		return home + value[1:]
	}
	return value
}

func evaluateDenyPatterns(command string, patterns []string, env GuardEnv) GuardDecision {
	for _, raw := range patterns {
		pattern := normalizeDenyPattern(raw, env.Home)
		if pattern == "" || isPathAwareDenyPattern(pattern) {
			continue
		}
		if matchShellGlob(pattern, command) {
			return denyGuard("native deny pattern=" + raw)
		}
	}
	return allowGuard()
}

// normalizeDenyPattern unwraps the Claude `Bash(...)` rule vocabulary and
// resolves home references so patterns compare against raw command text.
func normalizeDenyPattern(raw, home string) string {
	pattern := raw
	if strings.HasPrefix(pattern, "Bash(") && strings.HasSuffix(pattern, ")") {
		pattern = strings.TrimSuffix(strings.TrimPrefix(pattern, "Bash("), ")")
	}
	if home != "" {
		pattern = strings.ReplaceAll(pattern, "$HOME", home)
		pattern = strings.ReplaceAll(pattern, "~", home)
	}
	return pattern
}

// isPathAwareDenyPattern reports whether a pattern targets a command family
// already decided by resolved paths. Matching those textually would deny the
// approved ephemeral cases the path check exists to allow.
func isPathAwareDenyPattern(pattern string) bool {
	for _, family := range []string{"rm ", "find ", "truncate "} {
		if strings.HasPrefix(pattern, family) || strings.HasPrefix(pattern, "sudo "+family) {
			return true
		}
	}
	return false
}

// matchShellGlob applies shell wildcard semantics, in which `*` spans any
// characters including separators. Path-oriented matching would under-match
// command text.
func matchShellGlob(pattern, value string) bool {
	patternRunes, valueRunes := []rune(pattern), []rune(value)
	patternIndex, valueIndex := 0, 0
	starPattern, starValue := -1, 0
	for valueIndex < len(valueRunes) {
		if patternIndex < len(patternRunes) {
			switch patternRunes[patternIndex] {
			case '*':
				starPattern, starValue = patternIndex, valueIndex
				patternIndex++
				continue
			case '?':
				patternIndex++
				valueIndex++
				continue
			case '[':
				if next, ok := matchGlobClass(patternRunes, patternIndex, valueRunes[valueIndex]); ok {
					patternIndex = next
					valueIndex++
					continue
				}
			default:
				if patternRunes[patternIndex] == valueRunes[valueIndex] {
					patternIndex++
					valueIndex++
					continue
				}
			}
		}
		if starPattern < 0 {
			return false
		}
		starValue++
		valueIndex = starValue
		patternIndex = starPattern + 1
	}
	for patternIndex < len(patternRunes) && patternRunes[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(patternRunes)
}

// matchGlobClass evaluates the bracket expression opening at open against one
// character, returning the index just past the closing bracket.
func matchGlobClass(pattern []rune, open int, value rune) (int, bool) {
	index := open + 1
	negated := false
	if index < len(pattern) && (pattern[index] == '!' || pattern[index] == '^') {
		negated = true
		index++
	}
	matched, first := false, true
	for index < len(pattern) && (pattern[index] != ']' || first) {
		first = false
		if index+2 < len(pattern) && pattern[index+1] == '-' && pattern[index+2] != ']' {
			if value >= pattern[index] && value <= pattern[index+2] {
				matched = true
			}
			index += 3
			continue
		}
		if pattern[index] == value {
			matched = true
		}
		index++
	}
	if index >= len(pattern) {
		return 0, false
	}
	return index + 1, matched != negated
}

// splitShellTokens lexes command text with POSIX quoting rules. It reports an
// error for unterminated quotes and dangling escapes so a command that cannot
// be understood is never treated as understood.
func splitShellTokens(command string) ([]string, error) {
	tokens := make([]string, 0, 8)
	var current strings.Builder
	started := false
	flush := func() {
		if started {
			tokens = append(tokens, current.String())
			current.Reset()
			started = false
		}
	}
	runes := []rune(command)
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		switch {
		case char == ' ' || char == '\t' || char == '\n' || char == '\r':
			flush()
		case strings.ContainsRune(shellPunctuation, char):
			flush()
			var operator strings.Builder
			for index < len(runes) && strings.ContainsRune(shellPunctuation, runes[index]) {
				operator.WriteRune(runes[index])
				index++
			}
			index--
			tokens = append(tokens, operator.String())
		case char == '\'':
			index++
			start := index
			for index < len(runes) && runes[index] != '\'' {
				index++
			}
			if index >= len(runes) {
				return nil, errors.New("no closing quotation")
			}
			current.WriteString(string(runes[start:index]))
			started = true
		case char == '"':
			index++
			for index < len(runes) && runes[index] != '"' {
				if runes[index] == '\\' && index+1 < len(runes) && strings.ContainsRune(`"\$`+"`", runes[index+1]) {
					current.WriteRune(runes[index+1])
					index += 2
					continue
				}
				current.WriteRune(runes[index])
				index++
			}
			if index >= len(runes) {
				return nil, errors.New("no closing quotation")
			}
			started = true
		case char == '\\':
			if index+1 >= len(runes) {
				return nil, errors.New("no escaped character")
			}
			index++
			current.WriteRune(runes[index])
			started = true
		default:
			current.WriteRune(char)
			started = true
		}
	}
	flush()
	return tokens, nil
}

func containsControlOperator(tokens []string) bool {
	for _, token := range tokens {
		if _, ok := shellControlOperators[token]; ok {
			return true
		}
	}
	return false
}

func containsToken(tokens []string, wanted string) bool {
	for _, token := range tokens {
		if token == wanted {
			return true
		}
	}
	return false
}

// tokenIndex finds a command by name, accepting an absolute or relative
// invocation such as /bin/rm.
func tokenIndex(tokens []string, name string) int {
	for index, token := range tokens {
		if token == name || strings.HasSuffix(token, "/"+name) {
			return index
		}
	}
	return -1
}

// removeTargets collects rm operands, honouring the `--` end-of-options marker.
func removeTargets(tokens []string) []string {
	targets := make([]string, 0, len(tokens))
	endOptions := false
	for _, token := range tokens {
		if !endOptions && token == "--" {
			endOptions = true
			continue
		}
		if !endOptions && strings.HasPrefix(token, "-") {
			continue
		}
		targets = append(targets, token)
	}
	return targets
}

// leadingOperands collects the paths a find invocation walks, which precede
// its first predicate.
func leadingOperands(tokens []string) []string {
	targets := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if strings.HasPrefix(token, "-") {
			break
		}
		targets = append(targets, token)
	}
	return targets
}

// truncateTargets collects truncate operands, skipping the size argument.
func truncateTargets(tokens []string) []string {
	targets := make([]string, 0, len(tokens))
	skip := false
	for _, token := range tokens {
		switch {
		case skip:
			skip = false
		case token == "-s" || token == "--size":
			skip = true
		case !strings.HasPrefix(token, "-"):
			targets = append(targets, token)
		}
	}
	return targets
}

// RunHookGuard performs one PreToolUse decision end to end and returns the
// process exit code Claude Code reads.
func RunHookGuard(input io.Reader, stderr io.Writer, patterns []string, env GuardEnv) int {
	data, err := io.ReadAll(input)
	if err != nil {
		return refuse(stderr, env, "", "unreadable hook input")
	}
	command, err := ExtractHookCommand(data)
	if err != nil {
		return refuse(stderr, env, "", err.Error())
	}
	env.appendLog(fmt.Sprintf("%s tool=Bash cmd=%q patterns=%d", guardTimestamp(), env.excerpt(command), len(patterns)))
	if decision := EvaluateGuard(command, patterns, env); decision.Denied() {
		return refuse(stderr, env, command, decision.Reason)
	}
	return GuardExitContinue
}

func refuse(stderr io.Writer, env GuardEnv, command, reason string) int {
	env.appendLog(fmt.Sprintf("%s BLOCKED cmd=%q reason=%q", guardTimestamp(), env.excerpt(command), reason))
	if stderr != nil {
		fmt.Fprintf(stderr, "vrooli-managed deny rule blocked this command: %s\n", reason)
	}
	return GuardExitDeny
}

func guardTimestamp() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }

func (e GuardEnv) excerpt(command string) string {
	limit := e.LogCommandMax
	if limit <= 0 {
		limit = defaultGuardLogCommandMax
	}
	flattened := strings.NewReplacer("\n", " ", "\r", " ").Replace(command)
	runes := []rune(flattened)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return flattened
}

// appendLog records one decision. Audit logging is best effort: a hook that
// cannot write its log must still return a decision.
func (e GuardEnv) appendLog(line string) {
	if strings.TrimSpace(e.LogPath) == "" {
		return
	}
	// Owner-only: this directory holds an audit record of refused commands.
	if os.MkdirAll(filepath.Dir(e.LogPath), 0o700) != nil {
		return
	}
	limit := e.LogMaxBytes
	if limit <= 0 {
		limit = defaultGuardLogMaxBytes
	}
	if info, err := os.Stat(e.LogPath); err == nil && info.Size() > limit {
		_ = os.Rename(e.LogPath, e.LogPath+".1")
	}
	file, err := os.OpenFile(e.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = io.WriteString(file, line+"\n")
}
