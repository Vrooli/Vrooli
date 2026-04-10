import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import MessageJumpList from "../components/MessageJumpList";
import type { ConversationEvent } from "../lib/api";

function makeEvent(overrides: Partial<ConversationEvent> & { id: string; sequence: number }): ConversationEvent {
  return {
    sessionId: "sess-1",
    source: "claude_hook",
    role: "assistant",
    text: `Message ${overrides.sequence}`,
    speechParagraphs: [],
    summarized: false,
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    ...overrides,
  };
}

describe("MessageJumpList", () => {
  const onSelect = vi.fn();
  const onClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders all messages in the list", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "User question" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant answer" }),
      makeEvent({ id: "e3", sequence: 3, role: "assistant", text: "Follow up", source: "codex" }),
    ];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e2")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e3")).toBeInTheDocument();
  });

  it("displays sequence numbers and role labels", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "Hello" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Hi", source: "claude_hook" }),
    ];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    expect(screen.getByTestId("msg-jump-item-e1").textContent).toContain("#1");
    expect(screen.getByTestId("msg-jump-item-e1").textContent).toContain("You");
    expect(screen.getByTestId("msg-jump-item-e2").textContent).toContain("#2");
    expect(screen.getByTestId("msg-jump-item-e2").textContent).toContain("Claude");
  });

  it("truncates long message text", () => {
    const longText = "A".repeat(100);
    const events = [makeEvent({ id: "e1", sequence: 1, text: longText })];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    const item = screen.getByTestId("msg-jump-item-e1");
    // Should contain truncated text with ellipsis
    expect(item.textContent).toContain("…");
    // Should not contain the full 100-char text
    expect(item.textContent?.length).toBeLessThan(longText.length);
  });

  it("clicking an item calls onSelect and onClose", () => {
    const events = [makeEvent({ id: "e1", sequence: 1, text: "Click me" })];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
    expect(onSelect).toHaveBeenCalledWith("e1");
    expect(onClose).toHaveBeenCalled();
  });

  it("highlights the focused event", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ];

    render(<MessageJumpList events={events} focusedEventId="e2" onSelect={onSelect} onClose={onClose} />);

    const focusedItem = screen.getByTestId("msg-jump-item-e2");
    expect(focusedItem.className).toContain("bg-wc-accent");
  });

  it("shows empty state when no events", () => {
    render(<MessageJumpList events={[]} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    expect(screen.getByText("No messages")).toBeInTheDocument();
  });

  it("Escape key calls onClose", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    fireEvent.keyDown(screen.getByTestId("msg-jump-list"), { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
