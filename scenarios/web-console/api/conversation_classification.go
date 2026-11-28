package main

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/sessionstore"
)

// conversationKindWork is the projection-only kind of a row that summarises
// the hidden work between two conversation rows. It is never persisted.
const conversationKindWork = "work"

// conversationVisibleKinds are the kinds the Messages view lists as rows; every
// other kind is work collapsed between them. The SQL projection and the
// in-memory repository both read this one list.
var conversationVisibleKinds = []string{
	string(agenttranscript.KindMessage),
	string(agenttranscript.KindAttachment),
	string(agenttranscript.KindNotice),
}

func isVisibleConversationKind(kind string) bool {
	if kind == "" {
		return true
	}
	for _, visible := range conversationVisibleKinds {
		if kind == visible {
			return true
		}
	}
	return false
}

// isSpokenReply reports whether a row is the session agent's own reply on
// its main thread: the only rows auto-summarised and played as they arrive.
func isSpokenReply(event ConversationEvent) bool {
	return event.Role == ConversationRoleAssistant && storedConversationKind(event.Kind) == string(agenttranscript.KindMessage) &&
		event.ThreadID == "" && (event.Author == "" || event.Author == string(agenttranscript.AuthorAgent))
}

// conversationSourceHarnesses maps capture sources (and the "<provider>_archive"
// family) to the harness whose injected shapes classify their text.
var conversationSourceHarnesses = []struct{ prefix, harness string }{
	{"claude", agenttranscript.HarnessClaude},
	{"prompt_submit_hook", agenttranscript.HarnessClaude},
	{"codex", agenttranscript.HarnessCodex},
	{"opencode", "opencode"},
	{"grok", agenttranscript.HarnessGrok},
	{"gemini", "gemini"},
	{"antigravity", "antigravity"},
	{"qwen", "qwen"},
}

func harnessForConversationSource(source string) string {
	for _, entry := range conversationSourceHarnesses {
		if strings.HasPrefix(source, entry.prefix) {
			return entry.harness
		}
	}
	return ""
}

// classifyConversationRow classifies a stored row from its source and text;
// it is the classification for capture paths that only carry text and for
// the backfill of rows stored before classification existed.
func classifyConversationRow(source, role, text string) agenttranscript.Classification {
	return agenttranscript.ClassifyText(harnessForConversationSource(source), role, text)
}

// ConversationWorkSpan summarises the rows hidden between two conversation
// rows of one thread: the agent's tool use, reasoning and injected context.
// FromSequence..ToSequence covers every sequence in the gap (including other
// threads' rows) so paging stays contiguous; the counts and times are this
// thread's own.
type ConversationWorkSpan struct {
	FromSequence int64     `json:"fromSequence"`
	ToSequence   int64     `json:"toSequence"`
	ToolCalls    int       `json:"toolCalls"`
	ToolResults  int       `json:"toolResults"`
	Reasoning    int       `json:"reasoning"`
	Context      int       `json:"context"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	EndedAt      time.Time `json:"endedAt,omitempty"`
}

func (w *ConversationWorkSpan) add(kind string, at time.Time) {
	switch agenttranscript.MessageKind(kind) {
	case agenttranscript.KindToolCall:
		w.ToolCalls++
	case agenttranscript.KindToolResult:
		w.ToolResults++
	case agenttranscript.KindReasoning:
		w.Reasoning++
	default:
		w.Context++
	}
	if at.IsZero() {
		return
	}
	if w.StartedAt.IsZero() || at.Before(w.StartedAt) {
		w.StartedAt = at
	}
	if at.After(w.EndedAt) {
		w.EndedAt = at
	}
}

// workEvent is the projection row standing for a span.
func (w ConversationWorkSpan) workEvent(sessionID, threadID string) ConversationEvent {
	span := w
	return ConversationEvent{
		ID: "work:" + threadID + ":" + strconv.FormatInt(w.FromSequence, 10), SessionID: sessionID, Role: ConversationRoleAssistant,
		Kind: conversationKindWork, ThreadID: threadID, Sequence: w.ToSequence, CreatedAt: w.EndedAt, OccurredAt: w.EndedAt,
		Work: &span, SpeechParagraphs: []string{},
	}
}

// ConversationThread is a registered agent thread.
type ConversationThread struct {
	ThreadID       string    `json:"threadId"`
	Kind           string    `json:"kind"`
	Label          string    `json:"label"`
	Description    string    `json:"description"`
	ParentThreadID string    `json:"parentThreadId"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ConversationThreadSummary is a thread plus what its rows add up to.
type ConversationThreadSummary struct {
	ConversationThread
	FirstSequence int64     `json:"firstSequence"`
	LastSequence  int64     `json:"lastSequence"`
	MessageCount  int       `json:"messageCount"`
	WorkCount     int       `json:"workCount"`
	StartedAt     time.Time `json:"startedAt"`
	EndedAt       time.Time `json:"endedAt"`
}

func conversationThreadFromRef(ref agenttranscript.ThreadRef, createdAt time.Time) ConversationThread {
	kind := string(ref.Kind)
	if kind == "" {
		kind = string(agenttranscript.ThreadSubagent)
	}
	return ConversationThread{
		ThreadID: ref.Key(), Kind: kind, Label: strings.TrimSpace(ref.Name),
		Description: strings.TrimSpace(ref.Description), ParentThreadID: strings.TrimSpace(ref.ParentID), CreatedAt: createdAt,
	}
}

// conversationThreadStore registers threads and summarises their rows.
type conversationThreadStore interface {
	UpsertThread(ctx context.Context, sessionID string, thread ConversationThread) error
	ListThreadSummaries(ctx context.Context, sessionID string) ([]ConversationThreadSummary, error)
}

// ConversationProjectionPage is one page of the Messages projection: the
// conversation rows of one thread interleaved with work rows, plus (for the
// main thread's newest page) the session's agent threads.
type ConversationProjectionPage struct {
	State          ConversationSessionState
	HasMore        bool
	OldestSequence int64
	Threads        []ConversationThreadSummary
}

// ConversationProjectionQuery selects one projection page. SinceSequence > 0
// returns everything after it; otherwise the page ends before BeforeSequence
// (or at the newest row) and holds at most Limit conversation rows.
type ConversationProjectionQuery struct {
	ThreadID       string
	Limit          int
	BeforeSequence int64
	SinceSequence  int64
}

type conversationProjector interface {
	ListProjectionPage(ctx context.Context, sessionID string, query ConversationProjectionQuery) (ConversationProjectionPage, error)
	ListWorkRange(ctx context.Context, sessionID, threadID string, from, to int64, previewChars int) ([]ConversationEvent, error)
}

// launchPromptAttributor names the program that launched a session when a
// row is that session's launch prompt.
type launchPromptAttributor interface {
	LaunchPromptOwner(ctx context.Context, sessionID string) (string, bool)
}

type mainThreadPromptLookup interface {
	HasMainThreadPrompt(ctx context.Context, sessionID string) (bool, error)
}

// sessionLaunchPrompts attributes the first main-thread prompt of a
// programmatic session to its owner: a program opened the session and typed
// that prompt, so showing it as "You" would be false. Later prompts stay the
// human's; Web Console cannot tell an operator's typing from a program's.
type sessionLaunchPrompts struct {
	sessions sessionstore.Store
	prompts  mainThreadPromptLookup
	mu       sync.Mutex
	settled  map[string]bool
}

func newSessionLaunchPrompts(sessions sessionstore.Store, prompts mainThreadPromptLookup) *sessionLaunchPrompts {
	if sessions == nil || prompts == nil {
		return nil
	}
	return &sessionLaunchPrompts{sessions: sessions, prompts: prompts, settled: make(map[string]bool)}
}

func (a *sessionLaunchPrompts) LaunchPromptOwner(ctx context.Context, sessionID string) (string, bool) {
	if a == nil {
		return "", false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.settled[sessionID] {
		return "", false
	}
	row, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return "", false
	}
	if row.Origin != sessionstore.OriginProgrammatic {
		a.settled[sessionID] = true
		return "", false
	}
	has, err := a.prompts.HasMainThreadPrompt(ctx, sessionID)
	if err != nil {
		return "", false
	}
	a.settled[sessionID] = true
	if has {
		return "", false
	}
	return launchOwnerLabel(row.Owner), true
}

// launchOwnerLabel turns a session owner into a readable author name. Owners
// that are opaque ids read as a generic program.
func launchOwnerLabel(owner string) string {
	owner = strings.TrimSpace(owner)
	if owner == "" || strings.Count(owner, "-") >= 4 || strings.HasPrefix(owner, "target:") {
		return ""
	}
	return owner
}
