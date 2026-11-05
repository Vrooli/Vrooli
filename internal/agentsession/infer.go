package agentsession

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

// Environment variables the owner is read from. They are the owners' own
// contracts: web-console sets WC_* on every pane, Agent Manager sets the
// identity token and the runner tag, and the bundled desktop runtime sets
// VROOLI_DESKTOP_MODE and VROOLI_DATA on everything it starts.
const (
	envWebConsoleSession = "WC_WEB_CONSOLE_SESSION_ID"
	envWebConsoleState   = "WC_SESSION_STATE_ROOT"
	envTmux              = "TMUX"
	envIdentityToken     = "VROOLI_AGENT_IDENTITY_TOKEN"
	envCodexHome         = "CODEX_HOME"
	envGrokHome          = "GROK_HOME"
	envDesktopMode       = "VROOLI_DESKTOP_MODE"
	envDesktopData       = "VROOLI_DATA"
	envAppDataDir        = "APP_DATA_DIR"
	envBundleRoot        = "BUNDLE_ROOT"
	envSSHTTY            = "SSH_TTY"
	envScreen            = "STY"
)

// agentTagEnvs are the <RUNNER>_AGENT_TAG keys Agent Manager's codecs set.
var agentTagEnvs = []string{"CLAUDE_CODE_AGENT_TAG", "CODEX_AGENT_TAG", "OPENCODE_AGENT_TAG", "ANTIGRAVITY_AGENT_TAG", "AGENT_TAG"}

// runDirPattern finds an Agent Manager run directory in a per-run agent home
// (…/agent-manager/runs/<uuid>/codex).
var runDirPattern = regexp.MustCompile(`(?:^|/)agent-manager/runs/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(?:/|$)`)

// Infer reads a session's owner from its environment and controlling
// terminal. Precedence is deliberate:
//
//  1. a web-console session id wins, because an operator's terminal inside a
//     desktop build of web-console also carries the desktop markers, and the
//     terminal must outlive the app (persistent sessions survive restarts);
//  2. a bundled desktop app, whose helpers must end with the app;
//  3. an Agent Manager run, by run id when one is present and by tag only as
//     evidence (a tag names many runs over time, so it is never a ref);
//  4. an operator terminal (a tty, a non-web-console tmux, ssh or screen);
//  5. unknown.
func Infer(env map[string]string, tty string) Owner {
	evidence := Evidence{}
	runID, runSource := runIdentity(env)
	evidence.RunID, evidence.RunIDSource = runID, runSource
	evidence.AgentTag = agentTag(env)

	if sessionID := value(env, envWebConsoleSession); sessionID != "" {
		evidence.WebConsoleSessionID = sessionID
		evidence.WebConsoleStateRoot = cleanPath(value(env, envWebConsoleState))
		evidence.TmuxSocket = tmuxSocket(value(env, envTmux))
		evidence.DesktopData = cleanPath(value(env, envDesktopData))
		return Owner{Kind: KindWebConsoleSession, Ref: sessionID, Evidence: evidence}
	}
	if desktopMode(env) {
		data := cleanPath(value(env, envDesktopData))
		if data == "" {
			data = cleanPath(value(env, envAppDataDir))
		}
		evidence.DesktopData = data
		evidence.BundleRoot = cleanPath(value(env, envBundleRoot))
		return Owner{Kind: KindDesktopApp, Ref: data, Evidence: evidence}
	}
	if runID != "" || evidence.AgentTag != "" {
		return Owner{Kind: KindAgentManagerRun, Ref: runID, Evidence: evidence}
	}
	if terminal := terminalEvidence(env, tty); terminal != "" {
		evidence.Terminal = terminal
		return Owner{Kind: KindTerminal, Ref: terminal, Evidence: evidence}
	}
	return Owner{Kind: KindUnknown, Evidence: evidence}
}

// desktopMode is the bundled runtime's own marker. A bundle root alongside an
// app data directory is the same runtime's other spelling; either is enough.
func desktopMode(env map[string]string) bool {
	if strings.EqualFold(value(env, envDesktopMode), "true") || value(env, envDesktopMode) == "1" {
		return true
	}
	return value(env, envBundleRoot) != "" && value(env, envAppDataDir) != ""
}

// runIdentity returns the Agent Manager run id the environment carries. The
// identity token's claims are read without verifying the signature: the id
// selects which run to ask Agent Manager about, and Agent Manager's answer is
// the authority, so an id that names no run makes the owner gone only when
// Agent Manager says so.
func runIdentity(env map[string]string) (string, string) {
	if token := value(env, envIdentityToken); token != "" {
		if id := tokenRunID(token); id != "" {
			return id, "identity_token"
		}
	}
	for _, key := range []string{envCodexHome, envGrokHome} {
		if match := runDirPattern.FindStringSubmatch(filepath.ToSlash(value(env, key))); len(match) == 2 {
			return strings.ToLower(match[1]), strings.ToLower(key)
		}
	}
	return "", ""
}

// tokenRunID decodes the run_id claim of an Agent Manager identity token
// (base64url(claims) "." base64url(mac)).
func tokenRunID(token string) string {
	payload, _, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || payload == "" {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(payload, "="))
	if err != nil {
		return ""
	}
	var claims struct {
		RunID string `json:"run_id"`
	}
	if json.Unmarshal(raw, &claims) != nil {
		return ""
	}
	return strings.TrimSpace(claims.RunID)
}

func agentTag(env map[string]string) string {
	for _, key := range agentTagEnvs {
		if tag := value(env, key); tag != "" {
			return tag
		}
	}
	return ""
}

// terminalEvidence names an operator terminal: a controlling tty, a tmux that
// is not web-console's (web-console's own panes are caught by the session id
// first), ssh, or screen.
func terminalEvidence(env map[string]string, tty string) string {
	if tty = strings.TrimSpace(tty); tty != "" {
		return tty
	}
	if socket := tmuxSocket(value(env, envTmux)); socket != "" {
		return "tmux:" + socket
	}
	if sshTTY := value(env, envSSHTTY); sshTTY != "" {
		return "ssh:" + sshTTY
	}
	if screen := value(env, envScreen); screen != "" {
		return "screen:" + screen
	}
	return ""
}

// tmuxSocket returns the socket path of a TMUX value (socket,pid,session).
func tmuxSocket(raw string) string {
	socket, _, _ := strings.Cut(strings.TrimSpace(raw), ",")
	return cleanPath(socket)
}

func value(env map[string]string, key string) string {
	if env == nil {
		return ""
	}
	return strings.TrimSpace(env[key])
}

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// ProcessFact is what the host reports about one process.
type ProcessFact struct {
	PID  int
	PPID int
	// StartTime is the kernel's start stamp (clock ticks since boot on
	// Linux). Equal pid and StartTime is the same process; a reused pid has a
	// different StartTime.
	StartTime string
	// TTY is the controlling terminal device, or "".
	TTY string
}

// maxAncestry bounds the parent walk; a process tree deeper than this is not
// a launch chain.
const maxAncestry = 32

// DesktopAppRoot walks a launch's ancestry and returns the topmost ancestor
// that belongs to the same desktop app instance: the bundled runtime sets
// VROOLI_DESKTOP_MODE and VROOLI_DATA on everything it starts, so the highest
// process still carrying the same VROOLI_DATA is the app's runtime. Its pid
// and start time are the owner identity; when it exits the app has exited.
func DesktopAppRoot(self int, dataDir string, process func(int) (ProcessFact, error), environment func(int) (map[string]string, error)) (ProcessFact, bool) {
	dataDir = cleanPath(dataDir)
	if dataDir == "" || process == nil || environment == nil {
		return ProcessFact{}, false
	}
	var root ProcessFact
	found := false
	fact, err := process(self)
	if err != nil {
		return ProcessFact{}, false
	}
	for depth := 0; depth < maxAncestry && fact.PPID > 1; depth++ {
		parent, err := process(fact.PPID)
		if err != nil {
			break
		}
		env, err := environment(parent.PID)
		if err != nil {
			break
		}
		if !desktopMode(env) || cleanPath(firstNonEmpty(value(env, envDesktopData), value(env, envAppDataDir))) != dataDir {
			break
		}
		root, found = parent, true
		fact = parent
	}
	return root, found
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
