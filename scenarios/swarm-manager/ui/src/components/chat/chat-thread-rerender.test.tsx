/**
 * Re-render cost of a live transcript.
 *
 * The session detail page polls every 3s. Before this suite existed, each poll
 * re-rendered every message in the thread, and each of those re-parsed its
 * markdown and remounted any mermaid diagram — a cost that scaled with
 * transcript length and produced a visible flicker on diagrams.
 *
 * These tests measure the thing that actually matters (how many message bodies
 * React re-renders per poll) rather than asserting on any particular
 * memoization technique, so the implementation is free to change as long as the
 * cost does not come back.
 */
import { render, screen } from "@testing-library/react";
import { useEffect, useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChatThread } from "./ChatThread";
import type { ChatMessageView } from "./chat-types";

// The TTS controller reaches for audio-tools; the thread only needs it to be a
// stable, unavailable controller for these tests.
vi.mock("../../hooks/useAgentMessageTTS", () => ({
  useAgentMessageTTS: () => STABLE_TTS,
}));
const STABLE_TTS = {
  speak: () => undefined,
  stop: () => undefined,
  isSpeaking: false,
  speakingMessageId: null,
  loadingMessageId: null,
  unavailable: true,
};

// Counts how many times each message body is rendered, keyed by message id.
const renderCounts = new Map<string, number>();

vi.mock("@vrooli/react-component-library/markdown-renderer/0/0.3.2", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => {
    renderCounts.set(content, (renderCounts.get(content) ?? 0) + 1);
    return <div data-testid="markdown">{content}</div>;
  },
}));

function makeMessages(count: number): ChatMessageView[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `msg-${index}`,
    role: index % 2 === 0 ? ("user" as const) : ("assistant" as const),
    content: `message ${index}`,
    createdAt: `2026-05-01T12:0${index}:00Z`,
  }));
}

/**
 * Stands in for SessionDetailsPage under polling: it re-renders on a timer and
 * hands ChatThread the *same* messages array each time, which is what the store
 * now guarantees for an unchanged session.
 */
function PollingThread({ messages, polls }: { messages: ChatMessageView[]; polls: number }) {
  const [poll, setPoll] = useState(0);
  useEffect(() => {
    if (poll >= polls) return undefined;
    const timer = window.setTimeout(() => setPoll((value) => value + 1), 100);
    return () => window.clearTimeout(timer);
  }, [poll, polls]);
  return <ChatThread messages={messages} testId="thread" data-poll={poll} />;
}

describe("ChatThread re-render cost", () => {
  beforeEach(() => {
    renderCounts.clear();
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  it("does not re-render message bodies when a poll changes nothing", async () => {
    const messages = makeMessages(20);
    render(<PollingThread messages={messages} polls={5} />);

    await vi.advanceTimersByTimeAsync(800);

    // One render per message for the initial mount, and nothing after.
    for (const message of messages) {
      expect(renderCounts.get(message.content)).toBe(1);
    }
  });

  it("renders only the new message when one arrives", async () => {
    const messages = makeMessages(10);
    const { rerender } = render(<ChatThread messages={messages} testId="thread" />);
    for (const message of messages) expect(renderCounts.get(message.content)).toBe(1);

    const arrival: ChatMessageView = {
      id: "msg-new",
      role: "assistant",
      content: "a fresh reply",
      createdAt: "2026-05-01T13:00:00Z",
    };
    rerender(<ChatThread messages={[...messages, arrival]} testId="thread" />);

    expect(renderCounts.get("a fresh reply")).toBe(1);
    // The existing ten are untouched: a new arrival must not cost a full
    // transcript re-render.
    for (const message of messages) {
      expect(renderCounts.get(message.content)).toBe(1);
    }
    expect(screen.getAllByTestId("markdown")).toHaveLength(11);
  });

  it("re-renders only the message whose content changed", async () => {
    const messages = makeMessages(6);
    const { rerender } = render(<ChatThread messages={messages} testId="thread" />);

    const edited = messages.map((message, index) =>
      index === 3 ? { ...message, content: "edited body" } : message,
    );
    rerender(<ChatThread messages={edited} testId="thread" />);

    expect(renderCounts.get("edited body")).toBe(1);
    expect(renderCounts.get("message 0")).toBe(1);
    expect(renderCounts.get("message 5")).toBe(1);
  });
});
