import { describe, expect, it } from "vitest";
import {
  attachStarterSuggestions,
  composerSeedForOpener,
  isUnfilledOpener,
  starterCardBadgeSpec,
  starterSuggestionsForKind,
  type StarterSuggestion,
} from "./session-starter-suggestions";
import type { AgentSessionKind } from "../../types";

const CREATABLE_KINDS: AgentSessionKind[] = ["meta_orchestration", "swarm_operations", "workflow_authoring"];

function allCards(): Array<{ kind: AgentSessionKind; card: StarterSuggestion }> {
  return CREATABLE_KINDS.flatMap((kind) =>
    starterSuggestionsForKind(kind).map((card) => ({ kind, card })),
  );
}

describe("archived operating-mode-authoring sessions", () => {
  it("does not offer new starter actions", () => {
    expect(starterSuggestionsForKind("operating_mode_authoring")).toEqual([]);
  });
});

describe("a card's text is split by audience", () => {
  // Three readers, three homes: `label` is menu text for the operator choosing,
  // `opener` is composer text for the operator typing, and the job instruction
  // is agent text owned by the server. The defect this guards against is any
  // two of them collapsing back into one string — which is how "Turn this idea
  // into goals and backlog items." once got sent as a message with no idea
  // attached.

  it("gives every card a label and a server-owned job id", () => {
    for (const { kind, card } of allCards()) {
      expect(card.label.trim(), `${kind}/${card.id} label`).not.toBe("");
      expect(card.jobId, `${kind}/${card.id} job id`).toBe(card.id);
    }
  });

  it("keeps labels scannable", () => {
    for (const { kind, card } of allCards()) {
      expect(card.label.length, `${kind}/${card.id} label is too long for a menu`).toBeLessThanOrEqual(80);
    }
  });

  it("keeps openers short, invitational, and distinct from the label", () => {
    for (const { kind, card } of allCards()) {
      if (card.opener === undefined) continue;
      const opener = card.opener;
      expect(opener.trim(), `${kind}/${card.id} opener`).toBe(opener);
      // An opener is a prompt to the operator, not a paragraph. Anything long
      // enough to state a situation is job text wearing the wrong hat.
      expect(opener.length, `${kind}/${card.id} opener is doing the job band's work`).toBeLessThanOrEqual(60);
      expect(opener.endsWith(":"), `${kind}/${card.id} opener does not invite an answer`).toBe(true);
      expect(opener, `${kind}/${card.id} reuses its label as its opener`).not.toBe(card.label);
    }
  });

  it("never phrases an opener as an instruction to the agent", () => {
    // This is the regression that would quietly undo the split: an opener that
    // tells the agent what to do puts agent-directed text in the operator's
    // voice, and duplicates instruction the server already owns.
    const agentDirected = /\b(recommend|return|propose|identify|reconcile|assess|triage|review|analy[sz]e)\b/i;
    for (const { kind, card } of allCards()) {
      if (card.opener === undefined) continue;
      expect(agentDirected.test(card.opener), `${kind}/${card.id} opener instructs the agent`).toBe(false);
    }
  });

  it("leaves send-ready cards without an opener", () => {
    // These answer entirely from the startup brief and the staleness signal.
    // Seeding them would invent a requirement they do not have and turn a
    // one-click send into a chore.
    const operations = starterSuggestionsForKind("swarm_operations");
    for (const id of ["operations-review", "operations-sweep-staleness"]) {
      const card = operations.find((entry) => entry.id === id);
      expect(card, `${id} is missing`).toBeDefined();
      expect(card!.opener, `${id} asks for input it does not need`).toBeUndefined();
    }
  });

  it("keeps an opener on every card that needs the operator's own material", () => {
    // Each of these is useless without something only the operator can supply,
    // so an empty composer would leave them guessing where to start.
    const needsMaterial: Array<[AgentSessionKind, string]> = [
      ["meta_orchestration", "meta-plan"],
      ["meta_orchestration", "meta-existing"],
      ["workflow_authoring", "workflow-author-method"],
      ["workflow_authoring", "workflow-author-friction"],
      ["workflow_authoring", "workflow-author-transition"],
      ["workflow_authoring", "workflow-author-scenario"],
    ];
    for (const [kind, id] of needsMaterial) {
      const card = starterSuggestionsForKind(kind).find((entry) => entry.id === id);
      expect(card?.opener, `${kind}/${id} has no opener`).toBeTruthy();
    }
  });
});

describe("composer seeding", () => {
  it("puts the caret on a blank line under the invitation", () => {
    expect(composerSeedForOpener("Here is the idea:")).toBe("Here is the idea:\n\n");
  });

  it("recognises a seed the operator has not answered yet", () => {
    const seed = composerSeedForOpener("Here is the idea:");
    expect(isUnfilledOpener("meta_orchestration", seed)).toBe(true);
    expect(isUnfilledOpener("meta_orchestration", "Here is the idea:")).toBe(true);
  });

  it("treats an answered invitation as the operator's own message", () => {
    const answered = `${composerSeedForOpener("Here is the idea:")}A session inbox for captures.`;
    expect(isUnfilledOpener("meta_orchestration", answered)).toBe(false);
  });

  it("does not claim an empty composer or another kind's opener", () => {
    expect(isUnfilledOpener("meta_orchestration", "")).toBe(false);
    expect(isUnfilledOpener("meta_orchestration", "   \n\n ")).toBe(false);
    // "Here is how I work:" belongs to workflow_authoring; a Plan Work session
    // must not silently discard it as one of its own unfilled seeds.
    expect(isUnfilledOpener("meta_orchestration", "Here is how I work:")).toBe(false);
    expect(isUnfilledOpener("workflow_authoring", "Here is how I work:")).toBe(true);
  });
});

describe("attachStarterSuggestions (attach-sheet view of the starter cards)", () => {
  const backlog = { type: "backlog_item" as const, ref: "fix/demo", title: "Fix flaky stats test" };

  it("keeps only cards whose hard requirements the attached entity satisfies", () => {
    const ids = attachStarterSuggestions("meta_orchestration", backlog).map((s) => s.id);
    expect(ids).toContain("meta-plan"); // requirement-free
    expect(ids).toContain("meta-backlog"); // requires backlog_item — satisfied
    expect(ids).toContain("meta-existing"); // optional-only requirements
    expect(ids).not.toContain("meta-image"); // image-gated: the sheet has no attachment tray
  });

  it("interpolates the entity title into matching cards and sorts them first", () => {
    const suggestions = attachStarterSuggestions("meta_orchestration", backlog);
    // Mutation-list proposals are intentionally first for an attached item;
    // the conversational backlog card remains the specific workflow-aware option.
    const backlogCard = suggestions.find((suggestion) => suggestion.id === "meta-backlog");
    expect(backlogCard?.specific).toBe(true);
    expect(backlogCard?.label).toBe('Plan follow-up work for "Fix flaky stats test".');
    const generic = suggestions.find((s) => s.id === "meta-plan");
    expect(generic?.specific).toBe(false);
    expect(generic?.label).toBe("Turn an idea into goals and backlog items.");
  });

  it("carries a card's opener through so the quick-started session seeds the same way", () => {
    const suggestions = attachStarterSuggestions("meta_orchestration", backlog);
    expect(suggestions.find((s) => s.id === "meta-plan")?.opener).toBe("Here is the idea:");
    // The entity is staged as a visible context chip, so a lens needs nothing
    // typed and must not ask for it.
    const lens = suggestions.find((s) => s.proposalFlavor === "mutation_list");
    expect(lens?.opener).toBeUndefined();
  });

  it("drops cards that require a different context type", () => {
    const ids = attachStarterSuggestions("meta_orchestration", { type: "goal", ref: "g1", title: "Ship v2" }).map((s) => s.id);
    // A goal offers its proposal lenses first, then the generic cards it
    // satisfies. Cards gated on a backlog item or an image drop out.
    expect(ids).not.toContain("meta-backlog");
    expect(ids).not.toContain("meta-image");
    expect(ids).toContain("meta-plan");
    expect(ids).toContain("meta-existing");
    expect(ids.slice(0, 2)).toEqual(["proposal-split", "proposal-merge"]);
  });

  it("ignores requirement filter keys — an explicitly attached run offers the recovery card", () => {
    const suggestions = attachStarterSuggestions("swarm_operations", { type: "execution", ref: "exec-1", title: "run-42" });
    expect(suggestions[0]?.id).toBe("operations-run");
    expect(suggestions[0]?.label).toBe('Review run "run-42" and recommend recovery.');
  });

  it("falls back to the ref for blank titles and truncates long ones", () => {
    const blank = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "fix/no-title", title: "  " });
    expect(blank.find((suggestion) => suggestion.id === "meta-backlog")?.label).toBe('Plan follow-up work for "fix/no-title".');

    const long = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "x", title: "y".repeat(90) });
    expect(long.find((suggestion) => suggestion.id === "meta-backlog")?.label).toContain(`${"y".repeat(67)}...`);
  });

  it("names the entity in every proposal lens label", () => {
    // The lens instruction — including "never apply it" — is server-owned job
    // text. What the sheet still owes the operator is a label that says which
    // entity they are about to act on.
    const lenses = attachStarterSuggestions("swarm_operations", { type: "goal", ref: "ship-v2", title: "Ship v2" })
      .filter((suggestion) => suggestion.proposalFlavor === "mutation_list");
    expect(lenses.length).toBeGreaterThan(0);
    for (const lens of lenses) {
      expect(lens.label, `${lens.id} label`).toContain('"Ship v2"');
      expect(lens.jobId, `${lens.id} job id`).toBeTruthy();
    }
  });
});

describe("staleness starter cards", () => {
  const cards = starterSuggestionsForKind("swarm_operations");
  const scoped = cards.find((card) => card.id === "operations-triage-staleness");
  const sweep = cards.find((card) => card.id === "operations-sweep-staleness");

  it("gates the scoped card on attached items so clicking it opens the picker", () => {
    // Previously every requirement was optional, so the card seeded the
    // composer with text about "the attached items" while nothing was
    // attached, and never offered a way to attach anything.
    expect(scoped?.requirements).toContainEqual({ kind: "context", type: "backlog_item" });
    expect(starterCardBadgeSpec(scoped!)).toEqual({ type: "backlog_item", filterKey: undefined, gating: true });
  });

  it("does not word the scoped card's label as if something were already attached", () => {
    // The label is read while choosing, before anything is attached.
    expect(scoped?.label).not.toMatch(/the attached/i);
  });

  it("offers an unscoped sweep that needs no attachments and previews the stale count", () => {
    expect(sweep?.requirements).toBeUndefined();
    expect(starterCardBadgeSpec(sweep!)).toEqual({
      type: "backlog_item",
      filterKey: "backlog_item_stale",
      gating: false,
    });
  });

  it("never disables the sweep card, because a zero count is still a valid answer", () => {
    expect(starterCardBadgeSpec(sweep!)?.gating).toBe(false);
  });

  it("says in its own copy that the agent picks the set", () => {
    // The whole point of the unscoped card is not having to select by hand,
    // and that promise has to be visible while choosing.
    expect(sweep?.detail).toMatch(/picks the set itself/i);
  });
});
