import { describe, expect, it } from "vitest";
import type { PlanCardData, PlanCardGroupData, PlanGateData } from "../types";
import {
  applySnoozeFilter,
  cardSnoozeKey,
  countCards,
  gateActionLabel,
  laterWaveSummary,
  outcomeGlyph,
  splitBeyondHorizon,
  waveBadgeLabel,
} from "./plan-presentation";

function card(id: string, wave: number): PlanCardData {
  return {
    id,
    cardType: "item",
    action: "run",
    itemKind: "fix",
    itemName: id,
    title: id,
    status: "ready",
    priority: 3,
    wave,
    initiative: "",
    effort: "",
    gate: null,
    outcome: "",
    finishedAt: "",
    executionId: "",
    unblocks: 0,
  };
}

function group(id: string, blockerKind: PlanCardGroupData["blockerKind"], cards: PlanCardData[]): PlanCardGroupData {
  return { id, label: id, blockerKind, gateId: "", blockerKeys: [], cards };
}

function gate(kind: PlanGateData["kind"], count = 1, suggested = ""): PlanGateData {
  return {
    id: `${kind}:backlog/fix/x`,
    kind,
    ownerType: "backlog",
    ownerKind: "fix",
    ownerName: "x",
    ownerTitle: "x",
    count,
    blocks: [],
    decidableSince: "",
    suggested,
  };
}

describe("splitBeyondHorizon", () => {
  it("keeps shallow cards and rolls deep cards into the horizon", () => {
    const groups = [
      group("a", "items", [card("near", 2), card("deep", 7)]),
      group("b", "items", [card("deeper", 9)]),
    ];
    const split = splitBeyondHorizon(groups, 5);
    expect(split.visible).toHaveLength(1);
    expect(split.visible[0]?.cards.map((c) => c.id)).toEqual(["near"]);
    expect(split.beyond.map((c) => c.id)).toEqual(["deep", "deeper"]);
  });

  it("never rolls up cycle groups", () => {
    const groups = [group("cycle", "cycle", [card("trapped", -1)])];
    const split = splitBeyondHorizon(groups, 5);
    expect(split.visible).toHaveLength(1);
    expect(split.beyond).toHaveLength(0);
  });

  it("returns groups unchanged when nothing is deep", () => {
    const groups = [group("a", "items", [card("x", 1)])];
    const split = splitBeyondHorizon(groups, 5);
    expect(split.visible[0]).toBe(groups[0]);
    expect(split.beyond).toHaveLength(0);
  });
});

describe("waveBadgeLabel", () => {
  it("labels waves honestly", () => {
    expect(waveBadgeLabel(0)).toBe("now");
    expect(waveBadgeLabel(3)).toBe("w3");
    expect(waveBadgeLabel(-1)).toBe("cycle");
  });
});

describe("gateActionLabel", () => {
  it("labels each gate kind", () => {
    expect(gateActionLabel(gate("decide", 3))).toBe("Answer 3 questions");
    expect(gateActionLabel(gate("decide", 1))).toBe("Answer 1 question");
    expect(gateActionLabel(gate("review"))).toBe("Review");
    expect(gateActionLabel(gate("classify", 2))).toBe("Classify (2)");
    expect(gateActionLabel(gate("workshop", 1, "finalize"))).toBe("Finalize");
    expect(gateActionLabel(gate("workshop", 1, "workshop"))).toBe("Workshop");
  });
});

describe("outcomeGlyph", () => {
  it("maps outcomes to glyphs", () => {
    expect(outcomeGlyph("ok")).toBe("✓");
    expect(outcomeGlyph("failed")).toBe("✗");
    expect(outcomeGlyph("needs_review")).toBe("◆");
    expect(outcomeGlyph("needs_followup")).toBe("⚠");
    expect(outcomeGlyph("")).toBe("•");
  });
});

describe("laterWaveSummary", () => {
  it("summarizes count and distinct waves", () => {
    const groups = [
      group("a", "items", [card("x", 1), card("y", 2)]),
      group("b", "items", [card("z", 2)]),
    ];
    expect(laterWaveSummary(groups)).toBe("3 in 2 waves");
  });

  it("handles the empty column", () => {
    expect(laterWaveSummary([])).toBe("nothing blocked");
  });
});

describe("countCards", () => {
  it("sums cards across groups", () => {
    expect(countCards([group("a", "items", [card("x", 1)]), group("b", "items", [])])).toBe(1);
  });
});

describe("cardSnoozeKey", () => {
  it("builds keys matching the snooze store shapes", () => {
    expect(cardSnoozeKey(card("thing", 0))).toBe("backlog:fix/thing");
    expect(cardSnoozeKey({ ...card("x", 0), executionId: "e1" })).toBe("execution:e1");
    expect(cardSnoozeKey({ ...card("x", 0), id: "capture/c1", itemKind: "", itemName: "" })).toBe("capture:c1");
  });
});

describe("applySnoozeFilter", () => {
  const groups = [group("a", "items", [card("snoozed-item", 1), card("active-item", 1)])];
  const snoozed = new Set(["backlog:fix/snoozed-item"]);

  it("hides snoozed cards by default", () => {
    const result = applySnoozeFilter(groups, snoozed, false);
    expect(result.groups[0]?.cards.map((c) => c.itemName)).toEqual(["active-item"]);
    expect(result.hiddenCount).toBe(1);
    expect(result.snoozedIds.size).toBe(0);
  });

  it("keeps snoozed cards dimmed when showSnoozed is on", () => {
    const result = applySnoozeFilter(groups, snoozed, true);
    expect(result.groups[0]?.cards).toHaveLength(2);
    expect(result.snoozedIds.has("snoozed-item")).toBe(true);
    expect(result.hiddenCount).toBe(0);
  });

  it("drops groups that become empty", () => {
    const only = [group("a", "items", [card("snoozed-item", 1)])];
    const result = applySnoozeFilter(only, snoozed, false);
    expect(result.groups).toHaveLength(0);
  });
});
