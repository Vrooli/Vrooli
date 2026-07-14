import { describe, expect, it } from "vitest";
import {
  attachStarterSuggestions,
  countableTypesForKind,
  starterCardBadgeSpec,
  starterSuggestionsForKind,
} from "./session-starter-suggestions";

describe("operating-mode-authoring starter suggestions", () => {
  const suggestions = starterSuggestionsForKind("operating_mode_authoring");
  const byId = new Map(suggestions.map((s) => [s.id, s]));

  it("leads with a warm describe-first card that needs no attached context", () => {
    const lead = suggestions[0];
    expect(lead?.id).toBe("mode-describe");
    expect(lead?.requirements ?? []).toHaveLength(0);
    expect(lead?.text.toLowerCase()).toContain("phase graph");
  });

  it("offers a reuse-first start-from card gated on an operating_mode", () => {
    const startFrom = byId.get("mode-start-from");
    expect(startFrom).toBeDefined();
    const spec = starterCardBadgeSpec(startFrom!);
    expect(spec).toEqual({ type: "operating_mode", filterKey: undefined, gating: true });
  });

  it("counts every gating context type its cards require", () => {
    const countable = countableTypesForKind("operating_mode_authoring");
    expect(countable).toContain("operating_mode");
    expect(countable).toContain("initiative");
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
    expect(suggestions[0]?.id).toBe("meta-backlog");
    expect(suggestions[0]?.specific).toBe(true);
    expect(suggestions[0]?.text).toBe('Plan follow-up work for "Fix flaky stats test".');
    const generic = suggestions.find((s) => s.id === "meta-plan");
    expect(generic?.specific).toBe(false);
    expect(generic?.text).toBe("Turn this idea into initiatives and backlog items.");
  });

  it("drops cards that require a different context type", () => {
    const ids = attachStarterSuggestions("meta_orchestration", { type: "goal", ref: "g1", title: "Ship v2" }).map((s) => s.id);
    expect(ids).toEqual(["meta-plan", "meta-existing"]);
  });

  it("ignores requirement filter keys — an explicitly attached run offers the recovery card", () => {
    const suggestions = attachStarterSuggestions("swarm_operations", { type: "execution", ref: "exec-1", title: "run-42" });
    expect(suggestions[0]?.id).toBe("operations-run");
    expect(suggestions[0]?.text).toBe('Review run "run-42" and recommend recovery.');
  });

  it("falls back to the ref for blank titles and truncates long ones", () => {
    const blank = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "fix/no-title", title: "  " });
    expect(blank[0]?.text).toBe('Plan follow-up work for "fix/no-title".');

    const long = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "x", title: "y".repeat(90) });
    expect(long[0]?.text).toContain(`${"y".repeat(67)}...`);
  });
});
