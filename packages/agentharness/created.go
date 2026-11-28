package agentharness

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Files and folders an agent run created are the agent's own scratch, so the
// delete guard may let it remove them without asking. No single source can say
// which paths a run created, so several independent resolvers each answer with
// their own certainty, and the guard loosens only for a path the combined
// answer vouches for sufficiently. Everything else goes through the guard's
// existing decision unchanged.
//
// Certainty, from the strongest source down:
//
//	Recorded (3) a change record names the path as added by this run's workspace
//	             sandbox, which saw every write the agent made.
//	Medium   (2) the harness transcript records a creating tool result (a Write
//	             that reported "created", an apply_patch "Add File").
//	Low      (1) the transcript shows a shell command (mkdir, touch) that would
//	             have created it, which may have failed or run elsewhere; or the
//	             file system says the path was born after the run started,
//	             which any process on the host could have caused.
//
// Two independent resolvers that agree raise the result one level. Birth time
// is corroboration only, because it cannot say whose file it is: it lifts an
// attributing resolver's answer at most to Medium, and alone it stays Low.
// Low alone never loosens anything.

// Certainty is how sure a resolver is that the run created a path.
type Certainty int

const (
	CertaintyNone     Certainty = 0
	CertaintyLow      Certainty = 1
	CertaintyMedium   Certainty = 2
	CertaintyRecorded Certainty = 3
)

func (c Certainty) String() string {
	switch c {
	case CertaintyLow:
		return "low"
	case CertaintyMedium:
		return "medium"
	case CertaintyRecorded:
		return "recorded"
	}
	return "none"
}

// Thresholds: the certainty each kind of verdict needs before it is loosened.
const (
	// NeedForFloorDefault applies where only the guard's default would ask or
	// deny: a folder tree in the repository, or a path in the home folder
	// outside every declared location.
	NeedForFloorDefault = CertaintyMedium
	// NeedForDeclaredAsk applies where an owner declares a location needs its
	// confirmation: only a recorded change record opens it.
	NeedForDeclaredAsk = CertaintyRecorded
)

// Vouch is one resolver's (or the combined) answer for a path.
type Vouch struct {
	Certainty Certainty
	// Source names the resolver or resolvers; Detail says what they saw.
	Source, Detail string
	corroborates   bool
}

// CreationQuery asks whether the run created Path. Recursive means the whole
// tree below it will go too, so every part of the tree must be vouched.
type CreationQuery struct {
	Path      string
	Recursive bool
}

// CreationResolver answers "did this run create this path?" from one source.
// It must answer false, never guess, when it cannot tell.
type CreationResolver interface {
	Name() string
	Vouch(CreationQuery) (Vouch, bool)
}

// Corroborator marks a resolver that cannot say which process made a path, only
// that the path is new. Its answer counts as agreement with an attributing
// resolver, but never raises that answer past Medium.
type Corroborator interface {
	CorroboratesOnly() bool
}

// CreationEvidence combines every resolver and applies the git guard.
type CreationEvidence struct {
	Resolvers []CreationResolver
	// Tracked reports whether git tracks the path (or, for a recursive query,
	// anything below it). known is false when it cannot tell; the guard then
	// refuses to vouch for a path inside a work tree. Nil means no check.
	Tracked func(path string, recursive bool) (tracked, known bool)
}

// Vouch is the combined answer: the strongest resolver, raised one level when a
// second independent resolver agrees. A path git tracks, or may track, is never
// vouched: a created-and-staged file is work, not scratch.
func (e CreationEvidence) Vouch(q CreationQuery) (Vouch, bool) {
	var vouches []Vouch
	for _, resolver := range e.Resolvers {
		if v, ok := resolver.Vouch(q); ok && v.Certainty > CertaintyNone {
			if c, is := resolver.(Corroborator); is && c.CorroboratesOnly() {
				v.corroborates = true
			}
			vouches = append(vouches, v)
		}
	}
	if len(vouches) == 0 {
		return Vouch{}, false
	}
	if e.Tracked != nil {
		if tracked, known := e.Tracked(q.Path, q.Recursive); tracked || !known {
			return Vouch{}, false
		}
	}
	attributing, corroborating := 0, 0
	var best, bestAttributed Vouch
	var sources []string
	for _, v := range vouches {
		sources = append(sources, v.Source)
		if v.Certainty > best.Certainty {
			best = v
		}
		if v.corroborates {
			corroborating++
			continue
		}
		attributing++
		if v.Certainty > bestAttributed.Certainty {
			bestAttributed = v
		}
	}
	sort.Strings(sources)
	switch {
	case attributing == 0:
		// Only "the path is new": nobody says whose it is.
	case attributing > 1 && bestAttributed.Certainty < CertaintyRecorded:
		best = bestAttributed
		best.Certainty++
	case corroborating > 0 && bestAttributed.Certainty < CertaintyMedium:
		best = bestAttributed
		best.Certainty = CertaintyMedium
	default:
		best = bestAttributed
	}
	best.Source = strings.Join(sources, "+")
	return best, true
}

// RunIdentity is what the guard knows about the session whose command it is
// deciding. Every field is optional: a resolver with nothing to work from
// simply does not vouch.
type RunIdentity struct {
	Runner string
	// Session is the Vrooli agent session identity (VROOLI_AGENT_SESSION);
	// HarnessSession is the coding agent's own session id.
	Session, HarnessSession string
	// Start is when the session began; the zero time means unknown.
	Start time.Time
	// TranscriptPath is a transcript the harness named in its hook payload.
	TranscriptPath string
	// ClaudeHome and CodexHome are the harness homes the session runs under;
	// a Web Console terminal session or an isolated Agent Manager run has its
	// own CODEX_HOME.
	ClaudeHome, CodexHome string
	// SandboxID, SandboxMerged and SandboxURL name the run's workspace sandbox.
	SandboxID, SandboxMerged, SandboxURL string
	WorkingDirectory                     string
}

// Environment variable names the identity is read from.
const (
	envAgentSession   = "VROOLI_AGENT_SESSION"
	envAgentStartedAt = "VROOLI_AGENT_STARTED_AT"
	envSandboxID      = "VROOLI_SANDBOX_ID"
	envSandboxMerged  = "VROOLI_SANDBOX_MERGED"
	envSandboxURL     = "WORKSPACE_SANDBOX_URL"
)

// Context keys a harness adapter sets on a ToolEvent so the guard can find the
// session's transcript.
const (
	ContextTranscriptPath = "transcript_path"
	ContextSessionID      = "session_id"
)

// RunIdentityFromEvent reads the session identity from the hook event and the
// environment. lookup nil means os.LookupEnv.
func RunIdentityFromEvent(event ToolEvent, lookup func(string) (string, bool)) RunIdentity {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	get := func(names ...string) string {
		for _, name := range names {
			if value, ok := lookup(name); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
		return ""
	}
	id := RunIdentity{
		Runner:           event.Runner,
		Session:          get(envAgentSession),
		HarnessSession:   get("CLAUDE_CODE_SESSION_ID", "CODEX_SESSION_ID", "CODEX_THREAD_ID"),
		ClaudeHome:       get("CLAUDE_CONFIG_DIR"),
		CodexHome:        get("CODEX_HOME"),
		SandboxID:        get(envSandboxID),
		SandboxMerged:    get(envSandboxMerged),
		SandboxURL:       get(envSandboxURL),
		WorkingDirectory: event.WorkingDirectory,
	}
	if event.Context != nil {
		if value := strings.TrimSpace(event.Context[ContextSessionID]); value != "" {
			id.HarnessSession = value
		}
		id.TranscriptPath = strings.TrimSpace(event.Context[ContextTranscriptPath])
	}
	if raw := get(envAgentStartedAt); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			id.Start = t.UTC()
		}
	}
	return id
}

// maxStartWindow bounds how far back a session start counts. A session that
// has run for days would otherwise vouch for anything any process on the host
// made in that time.
const maxStartWindow = 24 * time.Hour

// HostCreationEvidence builds the resolvers this host can run for the session:
// the workspace-sandbox record, birth time since the session began, and the
// session's transcript. Resolvers that have nothing to work from are left out.
func HostCreationEvidence(id RunIdentity, now time.Time) CreationEvidence {
	evidence := CreationEvidence{Tracked: GitTracked}
	if id.SandboxID != "" && id.SandboxMerged != "" {
		if changes, err := FetchSandboxChanges(context.Background(), id.SandboxURL, id.SandboxID); err == nil {
			evidence.Resolvers = append(evidence.Resolvers, NewSandboxResolver(id.SandboxMerged, changes))
		}
	}
	creations, started := TranscriptCreations(id)
	if len(creations) > 0 {
		evidence.Resolvers = append(evidence.Resolvers, NewTranscriptResolver(creations))
	}
	start := id.Start
	if start.IsZero() {
		start = started
	}
	if !start.IsZero() {
		evidence.Resolvers = append(evidence.Resolvers, NewBirthTimeResolver(start, now))
	}
	return evidence
}

// GitTracked reports whether git tracks path, or anything below it when
// recursive. A path outside every work tree is untracked and known. Any failure
// to ask git inside a work tree reads as unknown.
func GitTracked(path string, recursive bool) (tracked, known bool) {
	if _, err := exec.LookPath("git"); err != nil {
		return false, !pathInsideGitTree(path)
	}
	dir := path
	for dir != "/" && dir != "." {
		if info, err := os.Lstat(dir); err == nil && info.IsDir() {
			break
		}
		dir = filepath.Dir(dir)
	}
	inside, err := gitOutput(dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if exitCode(err) == 128 {
			return false, true // not inside a work tree
		}
		return false, false
	}
	if strings.TrimSpace(inside) != "true" {
		return false, true
	}
	if recursive {
		listed, err := gitOutput(dir, "ls-files", "--", path)
		if err != nil {
			return false, false
		}
		return strings.TrimSpace(listed) != "", true
	}
	if _, err := gitOutput(dir, "ls-files", "--error-unmatch", "--", path); err != nil {
		if exitCode(err) == 1 {
			return false, true // untracked
		}
		return false, false
	}
	return true, true
}

func gitOutput(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	return string(out), err
}

func exitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}

// pathInsideGitTree is the fallback when git is not installed: a `.git`
// entry in any parent means the path may be tracked, so it is unknown.
func pathInsideGitTree(path string) bool {
	for dir := filepath.Dir(path); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
	}
	return false
}
