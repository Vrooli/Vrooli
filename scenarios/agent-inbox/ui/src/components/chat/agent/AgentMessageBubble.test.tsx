import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AgentMessageBubble } from "./AgentMessageBubble";
import type { AgentEvent } from "../../../lib/api";

vi.mock("../../markdown/MarkdownRenderer", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}));

vi.mock("../../ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("../../ui/toast", () => ({
  useToast: () => ({ addToast: vi.fn() }),
}));

function makeEvent(overrides: Partial<AgentEvent> = {}): AgentEvent {
  return {
    id: "evt-msg-1",
    type: "message",
    role: "assistant",
    content: "Test message",
    timestamp: new Date().toISOString(),
    sequence: 1,
    ...overrides,
  };
}

describe("AgentMessageBubble", () => {
  it("shows copy action in bubble mode", () => {
    render(<AgentMessageBubble event={makeEvent()} viewMode="bubble" />);
    expect(screen.getByTestId("agent-message-copy-evt-msg-1")).toBeInTheDocument();
  });

  it("shows copy action in compact mode", () => {
    render(<AgentMessageBubble event={makeEvent()} viewMode="compact" />);
    expect(screen.getByTestId("agent-message-copy-evt-msg-1")).toBeInTheDocument();
  });
});
