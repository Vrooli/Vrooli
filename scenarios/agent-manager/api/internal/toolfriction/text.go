package toolfriction

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// maxLine bounds a kept error line, notice or example command.
const maxLine = 200

// A result is read from its head and tail: errors, notices and truncation
// markers sit at the start (a refusal, a usage error) or the end (the last
// error of a long log) of a command's output.
const (
	ExcerptHead = 2000
	ExcerptTail = 1000
)

// excerpt is the part of a result's text the profiles read: all of it when
// short, else its head and tail.
func excerpt(text string) string {
	if len(text) <= ExcerptHead+ExcerptTail {
		return text
	}
	return strings.ToValidUTF8(text[:ExcerptHead], "") + "\n…\n" + strings.ToValidUTF8(text[len(text)-ExcerptTail:], "")
}

func clip(text string, n int) string {
	if len(text) <= n {
		return text
	}
	return strings.ToValidUTF8(text[:n], "")
}

// noiseLine is harness or CLI chatter that never names what went wrong:
// Codex exec envelopes, generic exit lines, rebuild notices, box-drawn
// banners and bare JSON punctuation.
var noiseLine = regexp.MustCompile(`(?i)^(script completed|script failed|wall time|output:|exit code:?|process exited with code|command exited with code|chunk id|original token count|total output lines|\+ |[{}\[\],]+$|[║╔╚═│┌└─]|✅|ℹ|⚠️? *if you are replacing)|(?i)^(runtime )?error: exit status \d+$`)

// errorLead starts the line a CLI prints for its error ("Error: …",
// "Runtime error: …", "fatal: …"); errorWord marks a line that names one.
var (
	errorLead = regexp.MustCompile(`(?i)^(error|fatal|panic|runtime error|usage error|failed|failure|exception|traceback)\b|\berror:`)
	errorWord = regexp.MustCompile(`(?i)\b(error|errors|failed|failure|fatal|panic|unknown|unrecognized|invalid|not found|no such|unavailable|refused|denied|cannot|can't|could not|timed out|saturated|over_subscribed|missing|required|exception|traceback|unauthorized|forbidden)\b`)
)

// jsonMessage reads the message of a JSON error line: "error": "…".
var jsonMessage = regexp.MustCompile(`^"(error|reason|detail|message|refused)"\s*:\s*"(.*)",?$`)

// firstErrorLine is the line of a failure's text that names the error: the
// first line a CLI prints as its error, else the first with an error word,
// else the first JSON error, reason, detail or message value, else the first
// line that is not noise, else the exit code.
func firstErrorLine(text string, exitCode *int) string {
	var worded, message, first string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(stripANSI(line))
		if line == "" || noiseLine.MatchString(line) || isNotice(line) {
			continue
		}
		if m := jsonMessage.FindStringSubmatch(line); m != nil {
			line = m[2]
			if message == "" {
				message = line
			}
		}
		if errorLead.MatchString(line) {
			return bound(line)
		}
		if worded == "" && errorWord.MatchString(line) {
			worded = line
		}
		if first == "" {
			first = line
		}
	}
	for _, line := range []string{worded, message, first} {
		if line != "" {
			return bound(line)
		}
	}
	if exitCode != nil {
		return "exit code " + strconv.Itoa(*exitCode)
	}
	return noErrorMessage
}

// noErrorMessage is the error line of a failure that printed none and has no
// exit code.
const noErrorMessage = "no error message"

// toolError is an error line that says a command could not do its job: the
// tool or its target was unavailable, refused, saturated, missing or timed
// out. A verify command that prints one never reached a verdict.
var toolError = regexp.MustCompile(`(?i)\b(unavailable|saturated|exhausted|over_subscribed|refused|denied|forbidden|unauthorized|timed out|timeout|not found|no such|cannot|can't|could not|unknown|invalid|panic)\b`)

// reportLine is a line of a verify command's report: a test result mark or a
// listed finding ("✓ src/x.test.ts > preserves unavailable paging", "- G2
// journey: …"). Its words describe what was checked, not the tool.
var reportLine = regexp.MustCompile(`^(?:[✓✔✗✘×√•]|(?:PASS|FAIL|ok)\b|--- |[-*] )`)

// reachedVerdict reports whether a verify command's non-zero exit is its
// verdict (a suite or lint that ran and failed): it was called with the right
// shape and its error line, if any, is a report line or names no tool error.
func reachedVerdict(errorLine string, shapeGuess bool) bool {
	return !shapeGuess && (silent(errorLine) || reportLine.MatchString(errorLine) || !toolError.MatchString(errorLine))
}

// A degradation is a line a command prints when it worked around something:
// a dependency was unavailable, degraded or refused and the command fell back
// (`agent-manager park unavailable … waiting inline instead`), or its
// providers were degraded. Prose, lists, tables and code a command merely
// prints (a skill, a document, a source file) are not degradations.
var (
	workaround    = regexp.MustCompile(`(?i)\b(instead|falling back|fell back|fallback|best-effort|continuing without|waiting inline)\b`)
	unavailable   = regexp.MustCompile(`(?i)\b(unavailable|degraded|failed|refused|down|unreachable|timed out)\b`)
	providersDown = regexp.MustCompile(`(?i)\bproviders? (were |was |is |are )?degraded\b`)
	quotedText    = regexp.MustCompile("^([-*|#>]|//|\\d+[.)] )|`")
)

// noticeLine is the first line of a call's output that reports a
// degradation, "" when none does. A failure's error line is never a notice. A
// summary that providers were degraded ("some providers degraded — see notes
// below") names them from the lines after it, so each degraded provider is
// its own finding with its own owner to tell.
func noticeLine(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line = strings.TrimSpace(stripANSI(line)); isNotice(line) {
			if providersDown.MatchString(line) {
				line = bound(line) + " [" + degradedProviders(lines[i+1:]) + "]"
			}
			return line
		}
	}
	return ""
}

// degradedProvider is a provider a CLI lists as degraded under such a summary
// (search-hub: "▸ agent-manager.runs — degraded: circuit unavailable").
var degradedProvider = regexp.MustCompile(`^▸\s*(\S+)\s+—\s+degraded\b`)

// unnamedProviders is the bracket of a providers-degraded summary whose output
// lists no degraded provider: the notes it promises ("see notes below") were
// not printed in what the agent read.
const unnamedProviders = "degraded provider not named in the output read"

// UnprintedNotes reports whether a notice line is a providers-degraded summary
// whose promised notes the output did not print.
func UnprintedNotes(line string) bool { return strings.HasSuffix(line, "["+unnamedProviders+"]") }

// degradedProviders names the providers the lines list as degraded, or says
// that the part of the output read names none.
func degradedProviders(lines []string) string {
	var names []string
	for _, line := range lines {
		if m := degradedProvider.FindStringSubmatch(strings.TrimSpace(stripANSI(line))); m != nil && !contains(names, m[1]) {
			names = append(names, m[1])
		}
	}
	if len(names) == 0 {
		return unnamedProviders
	}
	sort.Strings(names)
	return "degraded: " + strings.Join(names, ", ")
}

func isNotice(line string) bool {
	if line == "" || len(line) > maxLine*2 || strings.HasPrefix(line, "{") || strings.HasPrefix(line, `"`) || quotedText.MatchString(line) {
		return false
	}
	return providersDown.MatchString(line) || workaround.MatchString(line) && unavailable.MatchString(line)
}

// shapeError is a failure that says the command was called with the wrong
// shape: an unknown command, subcommand or flag, a missing argument, or a
// usage dump.
var shapeError = regexp.MustCompile(`unknown (command|subcommand|flag|option|argument|shorthand flag|verb)|unrecognized (option|argument|arguments|command|subcommand)|did you mean|flag provided but not defined|invalid (choice|command|subcommand|option|flag)|no such (command|subcommand|option|flag)|missing required|required flag|requires (at least |exactly |at most )?\d+ arg|accepts (at most |between )?\d+ arg|unexpected argument|too many arguments|usage: `)

// unknownToken is the command, subcommand or flag an "unknown command" error
// names, and suggestion the one a "did you mean" hint offers. They are read
// from the redacted error line, which keeps the quoted words.
var (
	unknownToken = regexp.MustCompile("(?i)(?:unknown|unsupported|unrecognized)(?: [a-z]+)? (?:command|subcommand|flag|option|verb)s?:? +[\"'`]?([^\\s\"'`()?,]+)")
	suggestion   = regexp.MustCompile("(?i)did you mean:? +[\"'`]?([^\\s\"'`()?,]+)")
)

// guessOf is the wrong command an error line names and the CLI's suggestion,
// "" for either it does not give.
func guessOf(line string) (token, suggested string) {
	if m := unknownToken.FindStringSubmatch(line); m != nil {
		token = m[1]
	}
	if m := suggestion.FindStringSubmatch(line); m != nil {
		suggested = m[1]
	}
	return token, suggested
}

// Normalisation replaces what differs between occurrences of one error or one
// command: ids, hashes, numbers, quoted values, paths, durations and ports.
var (
	uuidWord   = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	hexWord    = regexp.MustCompile(`\b[0-9a-fA-F]{12,}\b|\b(?:toolu|call|knw|run|ses|req)_[A-Za-z0-9]+\b`)
	quoted     = regexp.MustCompile("\"[^\"]*\"|'[^']*'|`[^`]*`")
	pathWord   = regexp.MustCompile(`(?:~|\.{1,2})?/[^\s:;,'")\]]+`)
	timeWord   = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ][\d:.]+Z?\b`)
	numberWord = regexp.MustCompile(`\b\d+(?:\.\d+)?(?:ms|s|m|h|k|kb|mb|gb|%)?\b`)
	spaces     = regexp.MustCompile(`\s+`)
)

// errorCode is a machine error code of three or more snake_case words
// (identity_refresh_unavailable, web_console_session_unverified): the same
// error whatever sentence each CLI version wraps it in.
var errorCode = regexp.MustCompile(`\b[a-z]+(?:_[a-z]+){2,}\b`)

// signature is the comparable form of an error line: its error code when it
// names one, else the normalised line.
func signature(line string) string {
	if code := errorCode.FindString(line); code != "" {
		return code
	}
	return normalise(line)
}

// normalise is the comparable form of a line: one error seen in many runs, or
// one status check run many times, normalises to one string.
func normalise(line string) string {
	line = uuidWord.ReplaceAllString(line, "<id>")
	line = timeWord.ReplaceAllString(line, "<time>")
	line = hexWord.ReplaceAllString(line, "<id>")
	line = quoted.ReplaceAllString(line, "<value>")
	line = pathWord.ReplaceAllString(line, "<path>")
	line = numberWord.ReplaceAllString(line, "<n>")
	return strings.TrimSpace(spaces.ReplaceAllString(line, " "))
}

// Redaction: secrets and long opaque values never leave the read model.
var (
	homePath    = regexp.MustCompile(`/home/[^/\s]+|/Users/[^/\s]+`)
	secretValue = regexp.MustCompile(`(?i)((?:token|password|secret|api[_-]?key|authorization|cookie)\s*[=:]\s*|bearer\s+)\S+`)
	opaqueRun   = regexp.MustCompile(`[A-Za-z0-9+_=-]{44,}`)
	ansi        = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
)

func redact(text string) string {
	text = homePath.ReplaceAllString(text, "~")
	text = secretValue.ReplaceAllString(text, "${1}<redacted>")
	text = opaqueRun.ReplaceAllString(text, "<redacted>")
	return bound(strings.TrimSpace(spaces.ReplaceAllString(text, " ")))
}

func stripANSI(text string) string { return ansi.ReplaceAllString(text, "") }

func bound(line string) string {
	if len(line) <= maxLine {
		return line
	}
	return strings.ToValidUTF8(line[:maxLine], "") + "…"
}
