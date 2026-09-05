import { describe, expect, it, vi } from "vitest";
import type { ConversationEvent } from "../../api/conversation";
import {
  MESSAGE_ACTIONS,
  actionPlacement,
  type MessageActionContext,
} from "./messageActions";

function event(role: ConversationEvent["role"]): ConversationEvent {
  return {
    id: "event-1",
    sessionId: "session-1",
    sequence: 1,
    source: "claude_hook",
    role,
    text: "Reusable message",
    speechParagraphs: ["Reusable message"],
    summarized: false,
    createdAt: "2026-08-28T00:00:00Z",
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
  };
}

function context(role: ConversationEvent["role"], overrides: Partial<MessageActionContext> = {}): MessageActionContext {
  return {
    event: event(role),
    sessionId: "session-1",
    readOnly: false,
    copied: false,
    isPlaintext: false,
    isAudioLoading: false,
    isTtsSpeaking: false,
    activeSpeakingEventId: null,
    summarizeLevel: "moderate",
    selectedVersion: "active",
    summarizingEventId: null,
    getSummarizeError: vi.fn(() => null),
    onClearSummarizeError: vi.fn(),
    onToggleSummarized: vi.fn(),
    onChangeLevel: vi.fn(),
    audioSettings: {
      volume: 1,
      isMuted: false,
      playbackRate: 1,
      capabilities: { canPause: true, canSeek: false, canAdjustSpeed: true, canAdjustVolume: true },
    },
    onSetPlaybackRate: vi.fn(),
    onSetVolume: vi.fn(),
    onSetMuted: vi.fn(),
    isMobile: false,
    onCopy: vi.fn(),
    onPlayFromHere: vi.fn(),
    onPlayEvent: vi.fn(),
    onToggleRenderMode: vi.fn(),
    ...overrides,
  };
}

const action = (id: string) => {
  const found = MESSAGE_ACTIONS.find((candidate) => candidate.id === id);
  if (!found) throw new Error(`Missing message action ${id}`);
  return found;
};

describe("MESSAGE_ACTIONS", () => {
  it("declares the complete action set in operator order", () => {
    expect(MESSAGE_ACTIONS.map(({ id }) => id)).toEqual([
      "copy",
      "read-from-here",
      "save-as-snippet",
      "handoff",
      "send-to-composer",
      "render-mode",
      "playback-mode",
      "audio-settings",
    ]);
  });

  it.each(["user", "assistant"] as const)("applies universal actions to %s messages", (role) => {
    const ctx = context(role);
    expect(action("copy").appliesTo(ctx)).toBe(true);
    expect(action("render-mode").appliesTo(ctx)).toBe(true);
  });

  it("limits TTS actions to writable non-user messages", () => {
    for (const id of ["read-from-here", "playback-mode", "audio-settings"]) {
      expect(action(id).appliesTo(context("assistant"))).toBe(true);
      expect(action(id).appliesTo(context("user"))).toBe(false);
      expect(action(id).appliesTo(context("assistant", { readOnly: true }))).toBe(false);
    }
  });

  it("shows capability-bound actions only when their callback is present", () => {
    expect(action("save-as-snippet").appliesTo(context("user"))).toBe(false);
    expect(action("handoff").appliesTo(context("assistant"))).toBe(false);
    expect(action("send-to-composer").appliesTo(context("assistant"))).toBe(false);

    expect(action("save-as-snippet").appliesTo(context("user", { onSaveAsSnippet: vi.fn() }))).toBe(true);
    expect(action("handoff").appliesTo(context("assistant", { onHandoff: vi.fn() }))).toBe(true);
    expect(action("send-to-composer").appliesTo(context("assistant", { onSendToComposer: vi.fn() }))).toBe(true);
  });

  it("places save inline for user messages and in overflow for other roles", () => {
    const save = action("save-as-snippet");
    expect(actionPlacement(save, context("user", { onSaveAsSnippet: vi.fn() }))).toBe("primary");
    expect(actionPlacement(save, context("assistant", { onSaveAsSnippet: vi.fn() }))).toBe("overflow");
  });

  it("runs copy and render behavior through the context", () => {
    const ctx = context("assistant");
    action("copy").run(ctx);
    action("render-mode").run(ctx);
    expect(ctx.onCopy).toHaveBeenCalledWith("event-1", "Reusable message");
    expect(ctx.onToggleRenderMode).toHaveBeenCalledWith("event-1");
  });
});
