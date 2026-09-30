import { describe, expect, it } from "vitest";
import { SessionActivitySource, SessionActivityState, type SessionActivity } from "@vrooli/proto-types/web-console/v1/sessions/sessions_pb";
import { decodeActivityPayload, decodeActivityProto } from "../sessionActivity";

describe("session activity decoding", () => {
  it("[REQ:P0-017i] reads a waiting prompt's hash, cancellable flag, and free-text hint from the hub payload", () => {
    const view = decodeActivityPayload({
      state: "waiting", source: "screen", confidence: 0.85, since: "2026-09-11T08:00:00Z", harness: "claude", harness_version: "2.1.268",
      prompt: {
        kind: "question", text: "Pick", options: [{ key: "1", label: "Red", selected: true }],
        answerable: true, hash: "h1", cancellable: true, free_text_hint: "Type something.",
      },
    });
    expect(view).toMatchObject({
      state: "waiting", source: "screen", harness: "claude", harnessVersion: "2.1.268",
      prompt: { kind: "question", text: "Pick", answerable: true, hash: "h1", cancellable: true, freeTextHint: "Type something." },
    });
    expect(view.prompt?.options).toEqual([{ key: "1", label: "Red", selected: true }]);
  });

  it("[REQ:P0-017e] reads unknown states and sources conservatively", () => {
    const view = decodeActivityPayload({ state: "bogus", source: "bogus" });
    expect(view.state).toBe("unknown");
    expect(view.source).toBe("output_clock");
    expect(view.prompt).toBeUndefined();
    expect(view.lastOutputAt).toBeUndefined();
  });

  it("[REQ:P0-017i] reads the sessions list's activity the same way", () => {
    const activity = {
      sessionId: "s1", state: SessionActivityState.WAITING, source: SessionActivitySource.HOOK, confidence: 0.9,
      since: "2026-09-11T08:00:00Z", lastOutputAt: "", harness: "opencode", harnessVersion: "",
      prompt: {
        kind: "permission", text: "bash git status", options: [{ key: "once", label: "Allow once", selected: false }],
        answerable: true, freeTextHint: "", hash: "h2", cancellable: false,
      },
    } as unknown as SessionActivity;
    const view = decodeActivityProto(activity);
    expect(view).toMatchObject({ state: "waiting", source: "hook", harness: "opencode", prompt: { kind: "permission", hash: "h2", answerable: true } });
    expect(view.prompt?.cancellable).toBeUndefined();
    expect(view.prompt?.freeTextHint).toBeUndefined();
  });
});
