import { describe, expect, it } from "vitest";
import { matchRules, matchesGlob, SCAN_EVENT_LIMIT, suggestionsForEvent } from "./captureRules";
import type { HandoffRuleDTO } from "../api/handoffrules";
import type { ConversationEvent } from "../api/conversation";

function event(id: string, text: string): ConversationEvent {
  return {
    id,
    sessionId: "s1",
    source: "claude",
    role: "assistant",
    text,
    speechParagraphs: [text],
    summarized: false,
    createdAt: "2026-08-27T00:00:00Z",
    sequence: Number(id.replace(/\D/g, "")) || 1,
    deliveryState: "delivered",
    ttsState: "idle",
    consumptionState: "unread",
  };
}

function rule(overrides: Partial<HandoffRuleDTO> = {}): HandoffRuleDTO {
  return {
    id: "r1",
    name: "Plan file",
    enabled: true,
    source: "file_path",
    pattern: "**/.vrooli/plans/*.md",
    surfaces: ["messages"],
    sort_order: 0,
    ...overrides,
  };
}

describe("matchesGlob", () => {
  it("matches a POSIX path", () => {
    expect(matchesGlob("**/.vrooli/plans/*.md", "/home/me/.vrooli/plans/a-plan.md")).toBe(true);
  });

  it("matches a Windows path in its own separator flavour", () => {
    expect(matchesGlob("**/.vrooli/plans/*.md", "C:\\Users\\me\\.vrooli\\plans\\a-plan.md")).toBe(true);
  });

  it("does not match a different directory", () => {
    expect(matchesGlob("**/.vrooli/plans/*.md", "/home/me/notes/a-plan.md")).toBe(false);
  });

  it("does not match a different extension", () => {
    expect(matchesGlob("**/.vrooli/plans/*.md", "/home/me/.vrooli/plans/a-plan.txt")).toBe(false);
  });

  it("keeps a single star inside one segment", () => {
    expect(matchesGlob("/tmp/*.md", "/tmp/a.md")).toBe(true);
    expect(matchesGlob("/tmp/*.md", "/tmp/nested/a.md")).toBe(false);
  });

  // A pattern written as a path must not be reinterpreted as a regular
  // expression by the operator who never wrote one.
  it("treats regular-expression syntax as literal text", () => {
    expect(matchesGlob("/tmp/a+b.md", "/tmp/a+b.md")).toBe(true);
    expect(matchesGlob("/tmp/a+b.md", "/tmp/aab.md")).toBe(false);
  });

  it("matches nothing for an empty glob", () => {
    expect(matchesGlob("", "/tmp/a.md")).toBe(false);
  });
});

describe("matchRules", () => {
  it("produces nothing when there are no rules", () => {
    expect(matchRules([], [event("e1", "See /home/me/.vrooli/plans/a.md")])).toEqual([]);
  });

  it("suggests a handoff for a path a session mentioned", () => {
    const out = matchRules([rule()], [event("e1", "The plan is at /home/me/.vrooli/plans/a-plan.md now")]);
    expect(out).toHaveLength(1);
    expect(out[0]).toMatchObject({
      ruleId: "r1",
      ruleName: "Plan file",
      eventId: "e1",
      payload: "/home/me/.vrooli/plans/a-plan.md",
    });
  });

  it("ignores a disabled rule", () => {
    const out = matchRules([rule({ enabled: false })], [event("e1", "at /home/me/.vrooli/plans/a.md")]);
    expect(out).toEqual([]);
  });

  it("ignores a rule with no pattern", () => {
    const out = matchRules([rule({ pattern: "" })], [event("e1", "at /home/me/.vrooli/plans/a.md")]);
    expect(out).toEqual([]);
  });

  it("takes the first capture group as the payload", () => {
    const out = matchRules(
      [rule({ source: "message_text", pattern: "wrote the plan to (\\S+)" })],
      [event("e1", "I wrote the plan to /tmp/plan.md just now")],
    );
    expect(out[0]?.payload).toBe("/tmp/plan.md");
  });

  it("takes the whole match when the pattern has no capture group", () => {
    const out = matchRules(
      [rule({ source: "message_text", pattern: "TODO:.*" })],
      [event("e1", "TODO: rename the thing")],
    );
    expect(out[0]?.payload).toBe("TODO: rename the thing");
  });

  it("scans no more than the recent window", () => {
    const events = Array.from({ length: SCAN_EVENT_LIMIT + 10 }, (_, i) =>
      event(`e${i}`, `plan at /home/me/.vrooli/plans/p${i}.md`));
    const out = matchRules([rule()], events);
    expect(out).toHaveLength(SCAN_EVENT_LIMIT);
    // The oldest events fall outside the window.
    expect(out.some((s) => s.eventId === "e0")).toBe(false);
    expect(out.some((s) => s.eventId === `e${SCAN_EVENT_LIMIT + 9}`)).toBe(true);
  });

  it("survives a pattern that will not compile", () => {
    const out = matchRules([rule({ source: "message_text", pattern: "([" })], [event("e1", "anything")]);
    expect(out).toEqual([]);
  });

  // An operator can write a pattern that backtracks badly. The input length is
  // bounded, so the scan stays bounded too.
  it("stays bounded on a pathological pattern", () => {
    const long = "a".repeat(60_000);
    const started = performance.now();
    matchRules([rule({ source: "message_text", pattern: "(a+)+$" })], [event("e1", `${long}b`)]);
    expect(performance.now() - started).toBeLessThan(3_000);
  });

  it("does not repeat the same suggestion for one event", () => {
    const out = matchRules([rule()], [event("e1", "/home/me/.vrooli/plans/a.md and /home/me/.vrooli/plans/a.md")]);
    expect(out).toHaveLength(1);
  });

  it("runs several rules over the same event", () => {
    const out = matchRules(
      [rule(), rule({ id: "r2", name: "Todo", source: "message_text", pattern: "TODO: (.+)" })],
      [event("e1", "TODO: read /home/me/.vrooli/plans/a.md")],
    );
    expect(out.map((s) => s.ruleId).sort()).toEqual(["r1", "r2"]);
  });
});

describe("suggestionsForEvent", () => {
  it("returns only the suggestions attached to one event", () => {
    const all = matchRules(
      [rule()],
      [event("e1", "at /home/me/.vrooli/plans/a.md"), event("e2", "at /home/me/.vrooli/plans/b.md")],
    );
    expect(suggestionsForEvent(all, "e2").map((s) => s.payload)).toEqual(["/home/me/.vrooli/plans/b.md"]);
  });
});
