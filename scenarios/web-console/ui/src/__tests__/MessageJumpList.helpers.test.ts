import { describe, it, expect } from "vitest";
import {
  applyFilter,
  formatRelativeTime,
  groupEventsByTurn,
  statusGlyphFor,
} from "../components/MessageJumpList.helpers";
import type { ConversationEvent } from "../api/conversation";

function makeEvent(overrides: Partial<ConversationEvent> & { id: string; sequence: number }): ConversationEvent {
  return {
    sessionId: "s",
    source: "claude_hook",
    role: "assistant",
    text: `m${overrides.sequence}`,
    speechParagraphs: [],
    summarized: false,
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    ...overrides,
  };
}

describe("groupEventsByTurn", () => {
  it("returns empty array for empty input", () => {
    expect(groupEventsByTurn([])).toEqual([]);
  });

  it("groups consecutive assistants under their preceding user event", () => {
    const events = [
      makeEvent({ id: "u1", sequence: 1, role: "user" }),
      makeEvent({ id: "a1", sequence: 2 }),
      makeEvent({ id: "a2", sequence: 3 }),
      makeEvent({ id: "u2", sequence: 4, role: "user" }),
      makeEvent({ id: "a3", sequence: 5 }),
    ];
    const turns = groupEventsByTurn(events);
    expect(turns).toHaveLength(2);
    const [first, second] = turns;
    expect(first?.user?.id).toBe("u1");
    expect(first?.assistants.map((e) => e.id)).toEqual(["a1", "a2"]);
    expect(second?.user?.id).toBe("u2");
    expect(second?.assistants.map((e) => e.id)).toEqual(["a3"]);
  });

  it("creates a synthetic null-user turn for leading assistant events", () => {
    const events = [
      makeEvent({ id: "a1", sequence: 1 }),
      makeEvent({ id: "u1", sequence: 2, role: "user" }),
      makeEvent({ id: "a2", sequence: 3 }),
    ];
    const turns = groupEventsByTurn(events);
    expect(turns).toHaveLength(2);
    const [first, second] = turns;
    expect(first?.user).toBeNull();
    expect(first?.assistants.map((e) => e.id)).toEqual(["a1"]);
    expect(second?.user?.id).toBe("u1");
  });

  it("handles trailing user with no replies", () => {
    const events = [
      makeEvent({ id: "u1", sequence: 1, role: "user" }),
      makeEvent({ id: "a1", sequence: 2 }),
      makeEvent({ id: "u2", sequence: 3, role: "user" }),
    ];
    const turns = groupEventsByTurn(events);
    expect(turns).toHaveLength(2);
    expect(turns[1]?.user?.id).toBe("u2");
    expect(turns[1]?.assistants).toEqual([]);
  });
});

describe("formatRelativeTime", () => {
  const now = new Date("2026-05-13T12:00:00Z");
  const at = (offsetMs: number) => new Date(now.getTime() - offsetMs).toISOString();

  it("returns 'just now' under 45 seconds", () => {
    expect(formatRelativeTime(at(0), now)).toBe("just now");
    expect(formatRelativeTime(at(44_000), now)).toBe("just now");
  });

  it("returns minutes between 45s and 1h", () => {
    expect(formatRelativeTime(at(60_000), now)).toBe("1m");
    expect(formatRelativeTime(at(30 * 60_000), now)).toBe("30m");
    expect(formatRelativeTime(at(59 * 60_000), now)).toBe("59m");
  });

  it("returns hours between 1h and 24h", () => {
    expect(formatRelativeTime(at(60 * 60_000), now)).toBe("1h");
    expect(formatRelativeTime(at(5 * 60 * 60_000), now)).toBe("5h");
  });

  it("returns a date string at 24h+", () => {
    const result = formatRelativeTime(at(48 * 60 * 60_000), now);
    expect(result).not.toBe("just now");
    expect(result).not.toMatch(/^\d+[mh]$/);
  });

  it("returns empty string for invalid input", () => {
    expect(formatRelativeTime("not-a-date", now)).toBe("");
  });
});

describe("statusGlyphFor", () => {
  it("maps ttsState=playing to playing", () => {
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "playing" })).glyph).toBe(
      "playing",
    );
  });

  it("maps ttsState=played to played", () => {
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "played" })).glyph).toBe(
      "played",
    );
  });

  it("maps ttsState=failed and rejected to failed", () => {
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "failed" })).glyph).toBe(
      "failed",
    );
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "rejected" })).glyph).toBe(
      "failed",
    );
  });

  it("falls back to consumptionState=listened as played when idle", () => {
    expect(
      statusGlyphFor(
        makeEvent({ id: "x", sequence: 1, ttsState: "idle", consumptionState: "listened" }),
      ).glyph,
    ).toBe("played");
  });

  it("defaults idle/unseen to unseen", () => {
    expect(
      statusGlyphFor(
        makeEvent({ id: "x", sequence: 1, ttsState: "idle", consumptionState: "unseen" }),
      ).glyph,
    ).toBe("unseen");
  });
});

describe("applyFilter", () => {
  const events = [
    makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
    makeEvent({ id: "b", sequence: 2, ttsState: "playing" }),
    makeEvent({ id: "c", sequence: 3, ttsState: "idle", consumptionState: "unseen" }),
    makeEvent({ id: "d", sequence: 4, ttsState: "failed" }),
    makeEvent({ id: "e", sequence: 5, ttsState: "rejected" }),
    makeEvent({ id: "f", sequence: 6, ttsState: "idle", consumptionState: "listened" }),
  ];

  it("all returns full list", () => {
    expect(applyFilter(events, "all")).toHaveLength(6);
  });

  it("unheard excludes played and listened", () => {
    const ids = applyFilter(events, "unheard").map((e) => e.id);
    expect(ids).toEqual(["b", "c", "d", "e"]);
  });

  it("failed includes only failed and rejected", () => {
    const ids = applyFilter(events, "failed").map((e) => e.id);
    expect(ids).toEqual(["d", "e"]);
  });
});
