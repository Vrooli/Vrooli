import { describe, expect, it } from "vitest";
import { ECHO_MATCH_WINDOW_MS, echoMatches, normalizeSentText, type EchoMatchEvent, type SentEcho } from "./echoMatch";

function mkEcho(overrides: Partial<SentEcho> = {}): SentEcho {
  return {
    id: "echo-1",
    sessionId: "s1",
    text: "run the tests",
    payload: "run the tests",
    attachmentCount: 0,
    sentAt: Date.parse("2026-09-11T08:00:00Z"),
    state: "staged",
    ...overrides,
  };
}

describe("normalizeSentText", () => {
  it("[REQ:P0-017f] trims, collapses whitespace, and drops a trailing carriage return", () => {
    expect(normalizeSentText("  run   the\ttests \r")).toBe("run the tests");
    expect(normalizeSentText("line one\nline two\r")).toBe("line one line two");
    expect(normalizeSentText("\r")).toBe("");
  });
});

describe("echoMatches", () => {
  const sentAt = Date.parse("2026-09-11T08:00:00Z");
  const echo = mkEcho();
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

  it("matches an attachment send the agent recorded with a chip instead of the path", () => {
    // Console sends the resolved path; the harness records it as the agent
    // rendered it. The user-text portion still aligns, so the row still clears.
    const withFile = mkEcho({ text: "describe this screenshot", payload: "describe this screenshot /tmp/upload_12.png", attachmentCount: 1 });
    expect(echoMatches(withFile, event({ text: "describe this screenshot [Image #1]" }))).toBe(true);
  });

  it("will not let a very short send sweep up an unrelated turn", () => {
    const short = mkEcho({ text: "y", payload: "y" });
    expect(echoMatches(short, event({ text: "yes please run everything" }))).toBe(false);
  });
});
