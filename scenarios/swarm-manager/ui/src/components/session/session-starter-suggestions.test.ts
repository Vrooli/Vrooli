import { describe, expect, it } from "vitest";
import {
  attachStarterSuggestions,
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

describe("label and prompt are two different jobs", () => {
  // A card carries menu text and composer text separately. When one string did
  // both, the label was sent as the message: "Turn this idea into goals and
  // backlog items." arrived with no idea attached and no indication that the
  // operator was meant to supply one.

  it("gives every card both a label and a prompt, and never reuses one as the other", () => {
    for (const { kind, card } of allCards()) {
      expect(card.label.trim(), `${kind}/${card.id} label`).not.toBe("");
      expect(card.prompt.trim(), `${kind}/${card.id} prompt`).not.toBe("");
      expect(card.prompt.trim(), `${kind}/${card.id} reuses its label as the prompt`).not.toBe(card.label.trim());
    }
  });

  it("keeps labels scannable and prompts substantive", () => {
    for (const { kind, card } of allCards()) {
      expect(card.label.length, `${kind}/${card.id} label is too long for a menu`).toBeLessThanOrEqual(80);
      // A prompt states the situation, the intent, and the shape of answer
      // wanted. A one-liner cannot do that, and a one-liner is exactly the
      // defect this split exists to prevent.
      expect(card.prompt.length, `${kind}/${card.id} prompt is a label in disguise`).toBeGreaterThan(120);
    }
  });

  it("only speaks about attached material on cards that require it", () => {
    // A card with no hard context requirement seeds its prompt immediately.
    // If that prompt says "the attached item", it refers to nothing.
    for (const { kind, card } of allCards()) {
      if (!/\bthe attached\b/i.test(card.prompt)) continue;
      // Any non-optional requirement counts: an image-gated card that says
      // "the attached image" is accurate, because the click opens the picker
      // before the prompt is seeded.
      const hasHardRequirement = (card.requirements ?? []).some((requirement) => !requirement.optional);
      expect(hasHardRequirement, `${kind}/${card.id} mentions attached material with no hard requirement`).toBe(true);
    }
  });

  it("ends an input-seeking prompt with a well-formed slot", () => {
    // Cards that need the operator's own material end with an invitation and a
    // blank line for it. A prompt that trails off in a colon with no slot
    // leaves the operator guessing where to type.
    for (const { kind, card } of allCards()) {
      if (!card.prompt.trimEnd().endsWith(":")) continue;
      expect(card.prompt, `${kind}/${card.id} has an invitation with no slot after it`).toMatch(/:\n\n$/);
    }
  });

  it("keeps send-ready cards free of a dangling invitation", () => {
    // These two answer entirely from the startup brief and the staleness
    // signal. Sending them unedited must be a complete request.
    const operations = starterSuggestionsForKind("swarm_operations");
    for (const id of ["operations-review", "operations-sweep-staleness"]) {
      const card = operations.find((entry) => entry.id === id);
      expect(card, `${id} is missing`).toBeDefined();
      expect(card!.prompt.trimEnd().endsWith(":"), `${id} asks for input it does not need`).toBe(false);
    }
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
    // The entity-specific prompt names the entity too, so the message reads as
    // being about it rather than about "the attached item".
    expect(backlogCard?.prompt).toContain('"Fix flaky stats test"');
    const generic = suggestions.find((s) => s.id === "meta-plan");
    expect(generic?.specific).toBe(false);
    expect(generic?.label).toBe("Turn an idea into goals and backlog items.");
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
    expect(suggestions[0]?.prompt).toContain('Run "run-42"');
  });

  it("falls back to the ref for blank titles and truncates long ones", () => {
    const blank = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "fix/no-title", title: "  " });
    expect(blank.find((suggestion) => suggestion.id === "meta-backlog")?.label).toBe('Plan follow-up work for "fix/no-title".');

    const long = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "x", title: "y".repeat(90) });
    expect(long.find((suggestion) => suggestion.id === "meta-backlog")?.label).toContain(`${"y".repeat(67)}...`);
  });

  it("gives every proposal lens a prompt that names the entity and forbids applying", () => {
    const lenses = attachStarterSuggestions("swarm_operations", { type: "goal", ref: "ship-v2", title: "Ship v2" })
      .filter((suggestion) => suggestion.proposalFlavor === "mutation_list");
    expect(lenses.length).toBeGreaterThan(0);
    for (const lens of lenses) {
      expect(lens.prompt, `${lens.id} prompt`).toContain('"Ship v2"');
      expect(lens.prompt.length, `${lens.id} prompt is a label in disguise`).toBeGreaterThan(120);
    }
  });
});

describe("staleness starter cards", () => {
  const cards = starterSuggestionsForKind("swarm_operations");
  const scoped = cards.find((card) => card.id === "operations-triage-staleness");
  const sweep = cards.find((card) => card.id === "operations-sweep-staleness");

  it("gates the scoped card on attached items so clicking it opens the picker", () => {
    // Previously every requirement was optional, so the card dropped text
    // referring to "the attached items" into the composer while nothing was
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

  it("tells the sweep agent to pick the set itself", () => {
    // The whole point of the unscoped card is not having to select by hand.
    expect(sweep?.prompt).toMatch(/do not ask me to pick/i);
  });
});
