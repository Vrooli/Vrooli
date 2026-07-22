import { describe, expect, it } from "vitest";
import {
  attachStarterSuggestions,
  starterSuggestionsForKind,
} from "./session-starter-suggestions";

describe("archived operating-mode-authoring sessions", () => {
  it("does not offer new starter actions", () => {
    expect(starterSuggestionsForKind("operating_mode_authoring")).toEqual([]);
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
    expect(backlogCard?.text).toBe('Plan follow-up work for "Fix flaky stats test".');
    const generic = suggestions.find((s) => s.id === "meta-plan");
    expect(generic?.specific).toBe(false);
    expect(generic?.text).toBe("Turn this idea into goals and backlog items.");
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
    expect(blank.find((suggestion) => suggestion.id === "meta-backlog")?.text).toBe('Plan follow-up work for "fix/no-title".');

    const long = attachStarterSuggestions("meta_orchestration", { type: "backlog_item", ref: "x", title: "y".repeat(90) });
    expect(long.find((suggestion) => suggestion.id === "meta-backlog")?.text).toContain(`${"y".repeat(67)}...`);
  });
});
