package harness

import (
	"context"
	"fmt"
)

// NewDefaultRegistry returns the BAS-native adapter surface. Third-party
// harnesses can be registered by deployments when their pinned integration is
// installed; they are never silently selected or made required here.
func NewDefaultRegistry() (*Registry, error) {
	return NewRegistry(Entry{
		Descriptor: Descriptor{
			ID:               "bas-typed-flow",
			Version:          "1",
			Summary:          "BAS-native typed observations and workflow actions through the current borrowed session.",
			ObservationModes: []string{"navigation", "screenshot"},
			ActionVocabulary: []string{"navigate", "click", "type", "select", "scroll", "wait", "assert", "extract", "screenshot"},
			EffectClasses:    []string{"observe", "page-navigation", "page-input", "page-state"},
			ReplayFidelity:   "typed-flow",
			InferenceRoles:   []string{"browser-observe", "browser-act"},
		},
		Probe: qualifyBASObservation,
	},
		Entry{
			Descriptor: Descriptor{
				ID: "accessibility-first", Version: "1",
				Summary:          "Reads the current BAS session through its owner-authorized accessibility tree; it never opens or closes a browser.",
				ObservationModes: []string{"accessibility-tree"},
				EffectClasses:    []string{"observe"},
				ReplayFidelity:   "observation-only",
				InferenceRoles:   []string{"browser-observe"},
			},
			Probe: qualifyAccessibilityFirst,
		},
		catalogueOnly("bas-vision-loop", "", "1", "BAS visual-navigation loop candidate; it must borrow the same session and obey the Agent Access action policy."),
		catalogueOnly("raw-cdp", "", "1", "Raw CDP adapter candidate; raw protocol authority is not exposed by the broker and this remains unsupported."),
		catalogueWithLimit("browser-use", "browser-use", "1.0.3", "Optional Browser Use Pydantic decision loop adapter; exact-pin Broker-only schema/task tests exist, pending live-session qualification and deployment integration.", "unsupported pending runtime integration and real target: BAS's optional exact-pin Python adapter uses `browser_use.agent.loop.build_agent` and `loop.run`, removes Python and finish_from_repl from the model-visible schema, and only exposes typed Broker task tools. Exact 1.0.3 in-process FunctionModel tests cover one-session snapshot/ref-click/snapshot, owner revoke, scope loss, and no service-Agent Browser start/stop or REPL construction. It is not installed in the BAS runtime or qualified against a real session/target"),
		catalogueWithLimit("playwright-mcp", "@playwright/mcp", "0.0.85", "Playwright MCP adapter; a pinned Broker-backed context façade and Agent Manager run-context task binding exist, pending real-session qualification.", "unsupported pending real target: exact 0.0.85 createConnection entrypoint has a BAS Page/Locator façade and four-tool allowlist. Production task transport binds a fixed pre-existing session and reads BAS URL, Agent Manager identity token, and browser grant from the Agent Manager-injected run environment; BAS revalidates these on every call. Exact-entrypoint fake-session tests cover read/ref/action, owner revoke, URL scope, ref expiry, and no-evaluate/lifecycle/selector escape. No real BAS task session or target was admitted"),
		catalogueWithLimit("playwright-cli", "@playwright/cli", "0.1.25", "Playwright CLI adapter; not qualified until every operation returns through the BAS Broker.", "unsupported: the upstream CLI controls a Playwright browser directly; a shared CDP endpoint would bypass Broker authorization and is not exposed"),
		catalogueOnly("stagehand", "@browserbasehq/stagehand", "4.3.0", "Stagehand adapter; must connect to the existing BAS session and keep inference roles explicit."),
	)
}

// qualifyAccessibilityFirst uses the current Broker-owned session's live AX
// surface. It qualifies only the read path; inference and actions are outside
// this adapter's current contract.
func qualifyAccessibilityFirst(ctx context.Context, session BorrowedSession) Qualification {
	observation, err := session.Observe(ctx, ObserveRequest{Mode: "accessibility-tree"})
	if err != nil {
		return Qualification{Outcome: OutcomeFailed, Error: fmt.Sprintf("accessibility observation failed: %v", err), Limits: []string{"task completion, inference, action, replay and real-target behavior were not exercised"}}
	}
	return Qualification{
		Outcome:  OutcomeQualified,
		Evidence: []string{fmt.Sprintf("borrowed-session accessibility tree succeeded (mode=%q)", observation.Mode)},
		Limits:   []string{"read-only accessibility observation only; no inference, actions, replay or real-target task was exercised"},
	}
}

// catalogueOnly makes candidate adapters discoverable while refusing to imply
// they can already attach to a borrowed session. Versions are explicit pins;
// no package is installed or launched by BAS as a side effect.
func catalogueOnly(id, upstream, version, summary string) Entry {
	return catalogueWithLimit(id, upstream, version, summary, "candidate is pinned and catalogued, but no same-session borrowed adapter is implemented")
}

func catalogueWithLimit(id, upstream, version, summary, limit string) Entry {
	return Entry{
		Descriptor: Descriptor{
			ID:               id,
			Version:          "1",
			Summary:          summary,
			ObservationModes: []string{"navigation", "screenshot", "accessibility"},
			ActionVocabulary: []string{"navigate", "click", "type", "scroll"},
			EffectClasses:    []string{"observe", "page-navigation", "page-input"},
			ReplayFidelity:   "unqualified",
			InferenceRoles:   []string{"browser-observe", "browser-act"},
			Upstream:         upstream,
			PinnedVersion:    version,
		},
		Probe: func(context.Context, BorrowedSession) Qualification {
			return Qualification{
				Outcome: OutcomeUnsupported,
				Limits:  []string{limit},
			}
		},
	}
}

// qualifyBASObservation proves access to the borrowed-session navigation
// observation path without causing page effects. DOM and accessibility
// snapshots are not yet exposed by the BAS session API. It intentionally does
// not claim to validate action fidelity or resistance to bot detection.
func qualifyBASObservation(ctx context.Context, session BorrowedSession) Qualification {
	observation, err := session.Observe(ctx, ObserveRequest{Mode: "navigation"})
	if err != nil {
		return Qualification{Outcome: OutcomeFailed, Error: fmt.Sprintf("navigation observation failed: %v", err), Limits: []string{"DOM/accessibility observation, action path and replay fidelity were not exercised"}}
	}
	return Qualification{
		Outcome:  OutcomeQualified,
		Evidence: []string{fmt.Sprintf("borrowed-session navigation observation succeeded (mode=%q)", observation.Mode)},
		Limits:   []string{"qualification exercised navigation observation only; DOM/accessibility snapshots, action replay fidelity and bot-detection behavior are not qualified"},
	}
}
