package agentsession

import (
	"encoding/json"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Owner kinds.
const (
	// KindAgentManagerRun is a session an Agent Manager run started; Ref is
	// the run id when the environment carries one.
	KindAgentManagerRun = "agent-manager-run"
	// KindWebConsoleSession is a session running inside a web-console
	// terminal session; Ref is the web-console session id.
	KindWebConsoleSession = "web-console-session"
	// KindDesktopApp is a session a bundled desktop app started; Ref is the
	// app instance's data directory (VROOLI_DATA).
	KindDesktopApp = "desktop-app"
	// KindTerminal is an interactive operator shell with no Vrooli owner.
	KindTerminal = "terminal"
	// KindUnknown is a session whose evidence names no owner.
	KindUnknown = "unknown"
)

// Owner sources say where an owner came from.
const (
	// SourceLaunch is an owner the launcher recorded when the session began.
	SourceLaunch = scenarioruntime.EditorLeaseOwnerLaunch
	// SourceBackfill is an owner inferred later and persisted on the lease.
	SourceBackfill = scenarioruntime.EditorLeaseOwnerBackfill
	// SourceInferred is an owner inferred for this reading only (a scope no
	// lease names has no row to persist it on).
	SourceInferred = "inferred"
)

// Web-console origins as the session store spells them.
const (
	OriginUI           = "ui"
	OriginProgrammatic = "programmatic"
	OriginRemote       = "remote"
	OriginUnspecified  = "unspecified"
)

// Evidence is what an owner was read from. Every field is a fact observed in
// the session's environment or process tree, or reported by the owner's
// authority; none is derived from a scope name.
type Evidence struct {
	// ReadFromPID is the process whose environment was read.
	ReadFromPID int `json:"read_from_pid,omitempty"`
	// RunID is the Agent Manager run id and RunIDSource where it came from
	// (identity_token claims, or the run directory in CODEX_HOME/GROK_HOME).
	RunID       string `json:"run_id,omitempty"`
	RunIDSource string `json:"run_id_source,omitempty"`
	// AgentTag is the <RUNNER>_AGENT_TAG Agent Manager sets on its runs.
	AgentTag string `json:"agent_tag,omitempty"`
	// WebConsoleSessionID, WebConsoleStateRoot and TmuxSocket locate a
	// web-console terminal session.
	WebConsoleSessionID string `json:"web_console_session_id,omitempty"`
	WebConsoleStateRoot string `json:"web_console_state_root,omitempty"`
	TmuxSocket          string `json:"tmux_socket,omitempty"`
	// WebConsoleOrigin and WebConsoleOwner are what web-console reported for
	// the session while it was alive. They are recorded because web-console
	// stops answering for a session once it is archived.
	WebConsoleOrigin string `json:"web_console_origin,omitempty"`
	WebConsoleOwner  string `json:"web_console_owner,omitempty"`
	// DesktopData and BundleRoot identify a bundled desktop app instance;
	// AppPID and AppStartTime are the app's runtime process when the launch
	// could see it (the topmost ancestor carrying the same VROOLI_DATA).
	DesktopData  string `json:"desktop_data,omitempty"`
	BundleRoot   string `json:"bundle_root,omitempty"`
	AppPID       int    `json:"app_pid,omitempty"`
	AppStartTime string `json:"app_start_time,omitempty"`
	// Terminal is the controlling terminal or multiplexer of an operator
	// shell.
	Terminal string `json:"terminal,omitempty"`
}

// Owner is what a session belongs to.
type Owner struct {
	Kind     string   `json:"kind"`
	Ref      string   `json:"ref,omitempty"`
	Source   string   `json:"source,omitempty"`
	Evidence Evidence `json:"evidence"`
}

// Known reports whether the owner names an owner kind that can be asked
// whether it is still there.
func (o Owner) Known() bool {
	switch o.Kind {
	case KindAgentManagerRun, KindWebConsoleSession, KindDesktopApp:
		return true
	default:
		return false
	}
}

// LeaseOwner converts an owner to the persisted lease shape.
func (o Owner) LeaseOwner() scenarioruntime.EditorLeaseOwner {
	evidence, err := json.Marshal(o.Evidence)
	if err != nil {
		evidence = []byte("{}")
	}
	return scenarioruntime.EditorLeaseOwner{Kind: o.Kind, Ref: o.Ref, EvidenceJSON: string(evidence), Source: o.Source}
}

// OwnerFromLease reads a persisted owner; ok is false when none is recorded.
func OwnerFromLease(owner scenarioruntime.EditorLeaseOwner) (Owner, bool) {
	kind := strings.TrimSpace(owner.Kind)
	if kind == "" {
		return Owner{}, false
	}
	out := Owner{Kind: kind, Ref: strings.TrimSpace(owner.Ref), Source: strings.TrimSpace(owner.Source)}
	if strings.TrimSpace(owner.EvidenceJSON) != "" {
		_ = json.Unmarshal([]byte(owner.EvidenceJSON), &out.Evidence)
	}
	return out, true
}
