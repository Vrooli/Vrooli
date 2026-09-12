import { describe, expect, it } from "vitest";
import { ECHO_MATCH_WINDOW_MS, echoMatches, normalizeSentText, type EchoMatchEvent } from "./echoMatch";

describe("normalizeSentText", () => {
  it("[REQ:P0-017f] trims, collapses whitespace, and drops a trailing carriage return", () => {
    expect(normalizeSentText("  run   the\ttests \r")).toBe("run the tests");
    expect(normalizeSentText("line one\nline two\r")).toBe("line one line two");
    expect(normalizeSentText("\r")).toBe("");
  });
});

describe("echoMatches", () => {
  const sentAt = Date.parse("2026-09-11T08:00:00Z");
  const echo = { id: "echo-1", sessionId: "s1", text: "run the tests", sentAt };
  const event = (overrides: Partial<EchoMatchEvent> = {}): EchoMatchEvent => ({
    sessionId: "s1",
    role: "user",
    text: "run  the tests\r",
    createdAt: "2026-09-11T08:00:20Z",
    ...overrides,
  });

  it("[REQ:P0-017f] matches the same normalised text in the same session within the window", () => {
    expect(echoMatches(echo, event())).toBe(true);
  });

  it("[REQ:P0-017f] does not match another session, an assistant turn, or different text", () => {
    expect(echoMatches(echo, event({ sessionId: "s2" }))).toBe(false);
    expect(echoMatches(echo, event({ role: "assistant" }))).toBe(false);
    expect(echoMatches(echo, event({ text: "run the build" }))).toBe(false);
  });

  it("[REQ:P0-017f] does not match outside the window", () => {
    const late = new Date(sentAt + ECHO_MATCH_WINDOW_MS + 1000).toISOString();
    expect(echoMatches(echo, event({ createdAt: late }))).toBe(false);
    const early = new Date(sentAt - ECHO_MATCH_WINDOW_MS - 1000).toISOString();
    expect(echoMatches(echo, event({ createdAt: early }))).toBe(false);
  });
});
