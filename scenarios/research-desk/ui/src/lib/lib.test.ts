import { describe, expect, it } from "vitest";

import type { Blocker } from "../api/launch";
import type { TopicSummary } from "../api/activity";
import { withRecommendations } from "./choices";
import { itemAnchor, itemKey, keyFromHash } from "./needsYou";
import { overlapPairs } from "./overlaps";
import { periodStart } from "./period";
import { matching, sortByResearchWrite } from "./topics";
import { withoutCommands } from "./words";

const blocker = (commissionId: string, recommended: Blocker["recommended"]): Blocker => ({ commissionId, state: "paused", topics: ["a"], message: "", choices: ["supersede", "alongside"], recommended, reason: "" });

describe("withRecommendations", () => {
  it("selects the desk's choice for new blockers only, and leaves the operator's alone", () => {
    expect(withRecommendations([blocker("p4", "supersede"), blocker("live", null)], {})).toEqual({ p4: "supersede", live: "" });
    expect(withRecommendations([blocker("p4", "supersede")], { p4: "alongside" })).toBeNull();
  });

  it("selects every recommended supersede, since a charter may supersede several", () => {
    expect(withRecommendations([blocker("p3", "supersede"), blocker("p4", "supersede")], {})).toEqual({ p3: "supersede", p4: "supersede" });
  });
});

describe("overlapPairs", () => {
  it("reads one row per pair with every topic it shares, keeping only pairs open now", () => {
    const overlaps = [
      { topic: "t1", commissionIds: ["b", "a"] },
      { topic: "t2", commissionIds: ["a", "b"] },
      { topic: "t1", commissionIds: ["a", "old"] },
    ] as Parameters<typeof overlapPairs>[0];
    expect(overlapPairs(overlaps, new Set(["a", "b"]))).toEqual([{ key: "a b", ids: ["a", "b"], topics: ["t1", "t2"] }]);
    expect(overlapPairs(overlaps)).toHaveLength(2);
  });
});

describe("needs-you anchors", () => {
  it("names an item by commission plus ID, global without one, and reads it back from the hash", () => {
    expect(itemKey({ commissionId: "c1", id: "AV4" })).toBe("c1/AV4");
    expect(itemAnchor({ commissionId: "", id: "G1" })).toBe("global%2FG1");
    expect(keyFromHash("#global%2FG1")).toBe("global/G1");
  });
});

describe("topics", () => {
  const topic = (name: string, seconds?: bigint) => ({ topic: name, title: { value: name.toUpperCase() }, lastResearchWrite: seconds === undefined ? undefined : { at: { value: { seconds } } } }) as unknown as TopicSummary;

  it("filters by name or title, and sorts the newest research write first, unwritten last by name", () => {
    const list = [topic("b"), topic("old", 1n), topic("new", 5n), topic("a")];
    expect(sortByResearchWrite(list).map((x) => x.topic)).toEqual(["new", "old", "a", "b"]);
    expect(matching(list, " NE ").map((x) => x.topic)).toEqual(["new"]);
    expect(matching(list, "")).toBe(list);
  });
});

describe("periodStart", () => {
  it("is undefined for the last look, and a minute-rounded start for a span", () => {
    expect(periodStart("look")).toBeUndefined();
    const now = Date.UTC(2026, 10, 12, 10, 30, 45);
    expect(periodStart("day", now)?.toISOString()).toBe("2026-11-11T10:30:00.000Z");
    expect(periodStart("week", now)?.toISOString()).toBe("2026-11-05T10:30:00.000Z");
  });
});

describe("withoutCommands", () => {
  it("drops a quoted command and its parentheses, keeping the sentence", () => {
    expect(withoutCommands("p1 is unknown; a relaunch is a new commission (`research-desk build --relaunch-of p1`)")).toBe("p1 is unknown; a relaunch is a new commission");
  });
});
