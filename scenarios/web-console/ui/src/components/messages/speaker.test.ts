import { describe, expect, it } from "vitest";
import { strings } from "../../consts/strings";
import { speakerKey, timeLabel } from "./speaker";

describe("speakerKey", () => {
  it("[REQ:P0-017c] names the user 'You' whatever the capture source", () => {
    expect(speakerKey({ role: "user", source: "claude_hook" })).toBe(strings.messagesPane.speaker.you);
    expect(speakerKey({ role: "user", source: "opencode_api" })).toBe(strings.messagesPane.speaker.you);
  });

  it.each([
    ["claude_hook", strings.messagesPane.speaker.claude],
    ["codex_tailer", strings.messagesPane.speaker.codex],
    ["grok_tailer", strings.messagesPane.speaker.grok],
    ["opencode_api", strings.messagesPane.speaker.opencode],
    ["something_new", strings.messagesPane.speaker.agent],
    ["", strings.messagesPane.speaker.agent],
  ])("[REQ:P0-017c] names an assistant captured by %s", (source, key) => {
    expect(speakerKey({ role: "assistant", source })).toBe(key);
  });
});

describe("timeLabel", () => {
  const now = new Date("2026-09-11T15:30:00");

  it("[REQ:P0-017c] shows the clock for a message from today", () => {
    const label = timeLabel("2026-09-11T14:41:00", now, "en-US");
    expect(label).toMatch(/2:41\s?PM/);
  });

  it("[REQ:P0-017c] shows a short date for an older message", () => {
    expect(timeLabel("2026-09-02T09:05:00", now, "en-US")).toBe("Sep 2");
  });

  it("returns an empty label for an unparseable timestamp", () => {
    expect(timeLabel("not a date", now, "en-US")).toBe("");
  });
});
