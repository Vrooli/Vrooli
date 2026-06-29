import { describe, it, expect } from "vitest";
import {
  assistantRoleLabelKey,
  availableSources,
  buildResults,
  computeExcerpt,
  DEFAULT_NAVIGATOR_STATE,
  detectBadges,
  formatRelativeTime,
  groupResults,
  noResultReason,
  normalizePreview,
  sourceIdFor,
  statusGlyphFor,
  type NavigatorState,
} from "../components/MessageJumpList.helpers";
import type { ConversationEvent } from "../api/conversation";

function makeEvent(
  overrides: Partial<ConversationEvent> & { id: string; sequence: number },
): ConversationEvent {
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

function state(overrides: Partial<NavigatorState> = {}): NavigatorState {
  return { ...DEFAULT_NAVIGATOR_STATE, ...overrides };
}

describe("assistantRoleLabelKey", () => {
  it("maps each conversation source to its agent label key", () => {
    expect(assistantRoleLabelKey("claude_hook")).toBe("messageJumpList.roleClaude");
    expect(assistantRoleLabelKey("codex_tailer")).toBe("messageJumpList.roleCodex");
    expect(assistantRoleLabelKey("opencode_api")).toBe("messageJumpList.roleOpenCode");
    expect(assistantRoleLabelKey("grok_tailer")).toBe("messageJumpList.roleGrok");
  });

  it("falls back to Codex for unknown sources", () => {
    expect(assistantRoleLabelKey("something_else")).toBe("messageJumpList.roleCodex");
  });
});

describe("sourceIdFor", () => {
  it("maps raw sources to stable ids with codex fallback", () => {
    expect(sourceIdFor("claude_hook")).toBe("claude");
    expect(sourceIdFor("opencode_api")).toBe("opencode");
    expect(sourceIdFor("grok_tailer")).toBe("grok");
    expect(sourceIdFor("codex_tailer")).toBe("codex");
    expect(sourceIdFor("mystery")).toBe("codex");
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
    expect(formatRelativeTime(at(59 * 60_000), now)).toBe("59m");
  });

  it("returns hours between 1h and 24h", () => {
    expect(formatRelativeTime(at(60 * 60_000), now)).toBe("1h");
  });

  it("returns empty string for invalid input", () => {
    expect(formatRelativeTime("not-a-date", now)).toBe("");
  });
});

describe("statusGlyphFor", () => {
  it("maps ttsState=playing/played/failed/rejected", () => {
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "playing" })).glyph).toBe("playing");
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "played" })).glyph).toBe("played");
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "failed" })).glyph).toBe("failed");
    expect(statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "rejected" })).glyph).toBe("failed");
  });

  it("falls back to consumptionState=listened as played when idle", () => {
    expect(
      statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "idle", consumptionState: "listened" })).glyph,
    ).toBe("played");
  });

  it("defaults idle/unseen to unseen", () => {
    expect(
      statusGlyphFor(makeEvent({ id: "x", sequence: 1, ttsState: "idle", consumptionState: "unseen" })).glyph,
    ).toBe("unseen");
  });
});

describe("normalizePreview", () => {
  it("strips markdown and collapses whitespace", () => {
    expect(normalizePreview("**Bold**   and `code`\n\nnext")).toBe("Bold and code next");
  });

  it("renders links as their text", () => {
    expect(normalizePreview("see [the docs](./README.md) now")).toBe("see the docs now");
  });
});

describe("detectBadges", () => {
  it("flags fenced and inline code", () => {
    expect(detectBadges(makeEvent({ id: "a", sequence: 1, text: "```\nx\n```" }))).toContain("code");
    expect(detectBadges(makeEvent({ id: "b", sequence: 2, text: "run `npm test`" }))).toContain("code");
  });

  it("flags file references in prose, inline code, and links", () => {
    expect(detectBadges(makeEvent({ id: "a", sequence: 1, text: "edit src/main.ts please" }))).toContain("fileReference");
    expect(detectBadges(makeEvent({ id: "b", sequence: 2, text: "see `config.json`" }))).toContain("fileReference");
    expect(detectBadges(makeEvent({ id: "c", sequence: 3, text: "[x](./a/b.go)" }))).toContain("fileReference");
  });

  it("does not flag ordinary prose as code or file reference", () => {
    const badges = detectBadges(makeEvent({ id: "a", sequence: 1, text: "Hello there, e.g. a normal sentence." }));
    expect(badges).not.toContain("code");
    expect(badges).not.toContain("fileReference");
  });

  it("flags long messages over the threshold", () => {
    expect(detectBadges(makeEvent({ id: "a", sequence: 1, text: "x".repeat(700) }))).toContain("long");
    expect(detectBadges(makeEvent({ id: "b", sequence: 2, text: "short" }))).not.toContain("long");
  });
});

describe("computeExcerpt", () => {
  it("returns a single non-match segment when no query", () => {
    const seg = computeExcerpt("hello world", "");
    expect(seg).toEqual([{ text: "hello world", match: false }]);
  });

  it("highlights a match at the beginning", () => {
    const seg = computeExcerpt("hello world", "hello");
    expect(seg[0]).toEqual({ text: "hello", match: true });
    expect(seg.map((s) => s.text).join("")).toBe("hello world");
  });

  it("highlights a match in the middle and adds leading ellipsis when windowed", () => {
    const preview = "a".repeat(80) + " needle " + "b".repeat(80);
    const seg = computeExcerpt(preview, "needle");
    expect(seg[0]?.text).toBe("…");
    expect(seg.some((s) => s.match && s.text === "needle")).toBe(true);
  });

  it("highlights a match at the end", () => {
    const seg = computeExcerpt("find the needle", "needle");
    expect(seg.some((s) => s.match && s.text === "needle")).toBe(true);
    expect(seg.map((s) => s.text).join("")).toContain("find the needle");
  });

  it("returns a leading slice when the query does not match the preview", () => {
    const seg = computeExcerpt("hello world", "zzz");
    expect(seg).toEqual([{ text: "hello world", match: false }]);
  });
});

describe("availableSources", () => {
  it("returns only sources present, in canonical order, ignoring user events", () => {
    const events = [
      makeEvent({ id: "u", sequence: 1, role: "user", source: "user" }),
      makeEvent({ id: "g", sequence: 2, source: "grok_tailer" }),
      makeEvent({ id: "c", sequence: 3, source: "claude_hook" }),
    ];
    expect(availableSources(events)).toEqual(["claude", "grok"]);
  });
});

describe("buildResults — filters", () => {
  const events = [
    makeEvent({ id: "u1", sequence: 1, role: "user", source: "user", text: "deploy the app" }),
    makeEvent({ id: "a1", sequence: 2, source: "claude_hook", text: "Running `make test` on src/app.ts", ttsState: "played" }),
    makeEvent({ id: "a2", sequence: 3, source: "codex_tailer", text: "deploy failed", ttsState: "failed" }),
    makeEvent({ id: "a3", sequence: 4, source: "grok_tailer", text: "summary", summarized: true, ttsState: "idle", consumptionState: "unseen" }),
  ];

  it("role=user keeps only user events", () => {
    const ids = buildResults(events, state({ role: "user" })).map((r) => r.event.id);
    expect(ids).toEqual(["u1"]);
  });

  it("role=assistant keeps only assistant events", () => {
    const ids = buildResults(events, state({ role: "assistant" })).map((r) => r.event.id);
    expect(ids).toEqual(["a1", "a2", "a3"]);
  });

  it("role=source:claude keeps only claude-sourced events", () => {
    const ids = buildResults(events, state({ role: "source:claude" })).map((r) => r.event.id);
    expect(ids).toEqual(["a1"]);
  });

  it("status=failed includes failed and rejected", () => {
    const withRejected = [...events, makeEvent({ id: "a4", sequence: 5, ttsState: "rejected" })];
    const ids = buildResults(withRejected, state({ status: "failed" })).map((r) => r.event.id);
    expect(ids).toEqual(["a2", "a4"]);
  });

  it("status=played includes played and listened", () => {
    const ids = buildResults(events, state({ status: "played" })).map((r) => r.event.id);
    expect(ids).toEqual(["a1"]);
  });

  it("status=unheard excludes played and listened", () => {
    const ids = buildResults(events, state({ status: "unheard" })).map((r) => r.event.id);
    expect(ids).toEqual(["u1", "a2", "a3"]);
  });

  it("status=summarized keeps only summarized events", () => {
    const ids = buildResults(events, state({ status: "summarized" })).map((r) => r.event.id);
    expect(ids).toEqual(["a3"]);
  });

  it("content=code keeps only events with code", () => {
    const ids = buildResults(events, state({ content: "code" })).map((r) => r.event.id);
    expect(ids).toEqual(["a1"]);
  });

  it("content=fileReference keeps only events referencing files", () => {
    const ids = buildResults(events, state({ content: "fileReference" })).map((r) => r.event.id);
    expect(ids).toEqual(["a1"]);
  });

  it("combines query and filters (AND)", () => {
    const ids = buildResults(events, state({ query: "deploy", role: "user" })).map((r) => r.event.id);
    expect(ids).toEqual(["u1"]);
  });
});

describe("buildResults — query + relevance", () => {
  const events = [
    makeEvent({ id: "a", sequence: 1, text: "alpha beta" }),
    makeEvent({ id: "b", sequence: 2, text: "beta beta beta" }),
    makeEvent({ id: "c", sequence: 3, text: "gamma" }),
  ];

  it("drops non-matching events when a query is set", () => {
    const ids = buildResults(events, state({ query: "beta" })).map((r) => r.event.id);
    expect(ids.sort()).toEqual(["a", "b"]);
  });

  it("matches metadata like sequence number and role", () => {
    const ids = buildResults(events, state({ query: "#2" })).map((r) => r.event.id);
    expect(ids).toEqual(["b"]);
  });

  it("relevance sort ranks more matches first; without query falls back to oldest", () => {
    const byRelevance = buildResults(events, state({ query: "beta", sort: "relevance" })).map((r) => r.event.id);
    expect(byRelevance).toEqual(["b", "a"]);

    const noQuery = buildResults(events, state({ sort: "relevance" })).map((r) => r.event.id);
    expect(noQuery).toEqual(["a", "b", "c"]);
  });

  it("populates excerpt highlight segments for matches", () => {
    const result = buildResults(events, state({ query: "alpha" }))[0];
    expect(result?.excerpt.some((s) => s.match)).toBe(true);
  });
});

describe("buildResults — sort", () => {
  const events = [
    makeEvent({ id: "a", sequence: 1 }),
    makeEvent({ id: "b", sequence: 2 }),
    makeEvent({ id: "c", sequence: 3 }),
  ];

  it("oldest keeps conversation order", () => {
    expect(buildResults(events, state({ sort: "oldest" })).map((r) => r.event.id)).toEqual(["a", "b", "c"]);
  });

  it("newest reverses by sequence", () => {
    expect(buildResults(events, state({ sort: "newest" })).map((r) => r.event.id)).toEqual(["c", "b", "a"]);
  });
});

describe("groupResults", () => {
  const events = [
    makeEvent({ id: "u1", sequence: 1, role: "user" }),
    makeEvent({ id: "a1", sequence: 2 }),
    makeEvent({ id: "a2", sequence: 3 }),
    makeEvent({ id: "u2", sequence: 4, role: "user" }),
    makeEvent({ id: "a3", sequence: 5 }),
  ];
  const results = buildResults(events, state());

  it("flat groups everything into one group preserving order", () => {
    const groups = groupResults(results, "flat");
    expect(groups).toHaveLength(1);
    expect(groups[0]?.items.map((r) => r.event.id)).toEqual(["u1", "a1", "a2", "u2", "a3"]);
  });

  it("turn groups assistants under their leading user, without losing identity", () => {
    const groups = groupResults(results, "turn");
    expect(groups).toHaveLength(2);
    expect(groups[0]?.leadUser?.event.id).toBe("u1");
    expect(groups[0]?.items.map((r) => r.event.id)).toEqual(["a1", "a2"]);
    expect(groups[1]?.leadUser?.event.id).toBe("u2");
    expect(groups[1]?.items.map((r) => r.event.id)).toEqual(["a3"]);
  });

  it("turn handles a leading run of assistant events with a null lead user", () => {
    const lead = buildResults(
      [makeEvent({ id: "a0", sequence: 0 }), ...events],
      state(),
    );
    const groups = groupResults(lead, "turn");
    expect(groups[0]?.leadUser).toBeNull();
    expect(groups[0]?.items.map((r) => r.event.id)).toEqual(["a0"]);
  });

  it("role groups users and assistants separately", () => {
    const groups = groupResults(results, "role");
    expect(groups.map((g) => g.roleLabel)).toEqual(["user", "assistant"]);
    expect(groups[0]?.items.map((r) => r.event.id)).toEqual(["u1", "u2"]);
    expect(groups[1]?.items.map((r) => r.event.id)).toEqual(["a1", "a2", "a3"]);
  });

  it("returns no groups for an empty result list", () => {
    expect(groupResults([], "turn")).toEqual([]);
  });
});

describe("noResultReason", () => {
  it("noMessages when the session is empty", () => {
    expect(noResultReason(0, state())).toBe("noMessages");
  });

  it("noSearchResults when only a query is active", () => {
    expect(noResultReason(5, state({ query: "x" }))).toBe("noSearchResults");
  });

  it("noFilterResults when only filters are active", () => {
    expect(noResultReason(5, state({ role: "user" }))).toBe("noFilterResults");
  });

  it("noResultsNarrow when both a query and filters are active", () => {
    expect(noResultReason(5, state({ query: "x", status: "failed" }))).toBe("noResultsNarrow");
  });
});
