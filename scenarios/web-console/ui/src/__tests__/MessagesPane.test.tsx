import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { useConversationStore } from "../stores/useConversationStore";
import type { ConversationEvent } from "../lib/api";

function makeEvent(overrides: Partial<ConversationEvent> & { id: string; sequence: number }): ConversationEvent {
  return {
    sessionId: "sess-1",
    source: "claude_hook",
    role: "assistant",
    text: `Message ${overrides.sequence}`,
    speechParagraphs: [`Message ${overrides.sequence}`],
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    ...overrides,
  };
}

const defaultProps = {
  sessionId: "sess-1",
  onSpeakFromHere: vi.fn(),
  onSpeakOne: vi.fn(),
  activeSpeakingEventId: null,
  isTtsSpeaking: false,
};

describe("MessagesPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useConversationStore.setState({ sessions: {}, viewModes: {} });
  });

  function seedEvents(events: ConversationEvent[]) {
    useConversationStore.setState({
      sessions: {
        "sess-1": {
          events,
          cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
          hydrated: true,
        },
      },
    });
  }

  it("renders play icons on each message card", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ];
    seedEvents(events);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-speak-from-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-speak-one-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-speak-from-e2")).toBeInTheDocument();
    expect(screen.getByTestId("msg-speak-one-e2")).toBeInTheDocument();
  });

  it("clicking 'read from here' calls onSpeakFromHere with correct event ID", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-speak-from-e1"));
    expect(defaultProps.onSpeakFromHere).toHaveBeenCalledWith("e1");
  });

  it("clicking 'read this one' calls onSpeakOne with event ID and text", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-speak-one-e1"));
    expect(defaultProps.onSpeakOne).toHaveBeenCalledWith("e1", "Hello world", ["Hello world"]);
  });

  it("active speaking event shows left border highlight", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ]);
    render(
      <MessagesPane
        {...defaultProps}
        activeSpeakingEventId="e2"
        isTtsSpeaking={true}
      />,
    );

    // Active card has accent left border
    const activeCard = screen.getByTestId("msg-card-e2");
    expect(activeCard.className).toContain("border-l-");

    // Inactive card does not
    const inactiveCard = screen.getByTestId("msg-card-e1");
    expect(inactiveCard.className).not.toContain("border-l-");
  });

  it("play button on active card triggers onSpeakFromHere (stop is in global bar)", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(
      <MessagesPane
        {...defaultProps}
        activeSpeakingEventId="e1"
        isTtsSpeaking={true}
      />,
    );

    // The play button always shows — stop lives in AudioPlayerBar
    fireEvent.click(screen.getByTestId("msg-speak-from-e1"));
    expect(defaultProps.onSpeakFromHere).toHaveBeenCalledWith("e1");
  });

  it("empty state renders without speaker icons", () => {
    seedEvents([]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId(/msg-speak-/)).toBeNull();
    expect(screen.getByText(/No conversation events/)).toBeInTheDocument();
  });

  it("displays correct source labels", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, source: "claude_hook" }),
      makeEvent({ id: "e2", sequence: 2, source: "codex" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByText("Claude Code")).toBeInTheDocument();
    expect(screen.getByText("Codex")).toBeInTheDocument();
  });
});
