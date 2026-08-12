package agentsessions

import "fmt"

// Session prompt sections follow a strict volatility gradient: universal, kind,
// job, volatile, then the task. Reference sections are wrapped in one XML
// context block so providers can cache every stable band before per-session and
// per-turn data changes. If a volatile section moves above a stable section, the
// first differing byte moves up with it and the cacheable prefix collapses.
//
// This mirrors the proven structure in
// `scenarios/prompt-manager/api/heartbeat/prompt_templates.go`. Do not invent a
// second prompt architecture; extend this registry instead.
//
// The defect this replaces: the session ID was emitted third, above every stable
// instruction, so no two sessions shared a prefix beyond roughly forty bytes.

type promptSectionScope int

// Scope values are ordered. The emitter sorts by this order, so the constant
// order here is the band order in the assembled prompt.
const (
	// promptScopeUniversal is byte-identical for every session of every kind.
	// Anything kind-specific belongs in a later band: a single varying byte
	// here destroys the shared prefix this band exists to create.
	promptScopeUniversal promptSectionScope = iota
	// promptScopeKind is byte-identical for every session of one kind.
	promptScopeKind
	// promptScopeJob varies by proposal target or starter job, and is stable
	// for the life of the session.
	promptScopeJob
	// promptScopeVolatile changes per session or per turn: identity, live
	// briefs, attached context, attachments.
	promptScopeVolatile
	// promptScopeTask is emitted outside the context block, in prose, so the
	// model sees the difference between material to consult and the job to do.
	promptScopeTask
)

const (
	promptSectionKindDoctrine       = "session-doctrine"
	promptSectionKindSubject        = "session-kind"
	promptSectionKindStarterJob     = "starter-job"
	promptSectionKindProposalTarget = "proposal-target"
	promptSectionKindLedgerWake     = "ledger-wake"
	promptSectionKindFallback       = "continuity-fallback"
	promptSectionKindIdentity       = "session-identity"
	promptSectionKindStartupBrief   = "startup-brief"
	promptSectionKindContext        = "attached-context"
	promptSectionKindImages         = "attached-images"
	promptSectionKindOperatorMsg    = "operator-message"
)

// promptSectionSpec describes a stable section identity emitted by the session
// prompt. The registry prevents a renderer from introducing an untracked
// section that clients and tests cannot reliably interpret.
type promptSectionSpec struct {
	Element string
	Scope   promptSectionScope
}

var promptSectionSpecs = map[string]promptSectionSpec{
	promptSectionKindDoctrine:       {Element: "session-doctrine", Scope: promptScopeUniversal},
	promptSectionKindSubject:        {Element: "session-kind", Scope: promptScopeKind},
	promptSectionKindStarterJob:     {Element: "starter-job", Scope: promptScopeJob},
	promptSectionKindProposalTarget: {Element: "proposal-target", Scope: promptScopeJob},
	promptSectionKindLedgerWake:     {Element: "ledger-wake", Scope: promptScopeVolatile},
	promptSectionKindFallback:       {Element: "continuity-fallback", Scope: promptScopeVolatile},
	promptSectionKindIdentity:       {Element: "session-identity", Scope: promptScopeVolatile},
	promptSectionKindStartupBrief:   {Element: "startup-brief", Scope: promptScopeVolatile},
	promptSectionKindContext:        {Element: "attached-context", Scope: promptScopeVolatile},
	promptSectionKindImages:         {Element: "attached-images", Scope: promptScopeVolatile},
	promptSectionKindOperatorMsg:    {Element: "operator-message", Scope: promptScopeTask},
}

// promptSection is one assembled band of the session prompt.
type promptSection struct {
	Kind    string
	Content string
	// Attrs are rendered on the section's opening tag, already escaped.
	Attrs string
}

// newPromptSection builds a section carrying its registered identity. An
// unregistered kind panics rather than emitting an unnamed block: a section the
// registry does not name is the drift this registry exists to prevent.
func newPromptSection(kind, attrs, content string) promptSection {
	if _, ok := promptSectionSpecs[kind]; !ok {
		panic(fmt.Sprintf("unregistered session prompt section kind %q", kind))
	}
	return promptSection{Kind: kind, Attrs: attrs, Content: content}
}

func promptSectionElement(kind string) string {
	spec, ok := promptSectionSpecs[kind]
	if !ok {
		panic(fmt.Sprintf("unregistered session prompt section kind %q", kind))
	}
	return spec.Element
}

func promptSectionScopeOf(kind string) promptSectionScope {
	spec, ok := promptSectionSpecs[kind]
	if !ok {
		panic(fmt.Sprintf("unregistered session prompt section kind %q", kind))
	}
	return spec.Scope
}
