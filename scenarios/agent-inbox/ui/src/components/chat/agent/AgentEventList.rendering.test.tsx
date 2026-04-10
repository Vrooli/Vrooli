/**
 * Tests for AgentEventList - Message rendering, errors, status, mixed events, compaction
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AgentEventList } from "./AgentEventList";
import type { AgentEvent } from "../../../lib/api";

// Mock child components
vi.mock("./AgentMessageBubble", () => ({
  default: ({ event }: { event: AgentEvent }) => (
    <div data-testid={`message-${event.id}`}>Message: {event.content}</div>
  ),
  AgentMessageBubble: ({ event }: { event: AgentEvent }) => (
    <div data-testid={`message-${event.id}`}>Message: {event.content}</div>
  ),
}));

vi.mock("./AgentToolCallCard", () => ({
  default: ({ event }: { event: AgentEvent; result?: AgentEvent }) => (
    <div data-testid={`tool-fallback-${event.id}`}>Tool: {event.tool_name}</div>
  ),
  AgentToolCallCard: ({ event }: { event: AgentEvent; result?: AgentEvent }) => (
    <div data-testid={`tool-fallback-${event.id}`}>Tool: {event.tool_name}</div>
  ),
}));

vi.mock("./AgentRawEventCard", () => ({
  default: ({ event }: { event: AgentEvent }) => (
    <div data-testid={`raw-${event.id}`}>Raw: {event.type}</div>
  ),
  AgentRawEventCard: ({ event }: { event: AgentEvent }) => (
    <div data-testid={`raw-${event.id}`}>Raw: {event.type}</div>
  ),
}));

vi.mock("./AgentCompactionCard", () => ({
  default: ({ event }: { event: AgentEvent }) => (
    <div data-testid={`compaction-${event.id}`}>Compaction: {event.content}</div>
  ),
  AgentCompactionCard: ({ event }: { event: AgentEvent }) => (
    <div data-testid={`compaction-${event.id}`}>Compaction: {event.content}</div>
  ),
}));

vi.mock("./tools", () => ({
  getToolComponent: (_toolName?: string, _runnerType?: string) => {
    return ({ event, result }: { event: AgentEvent; result?: AgentEvent }) => (
      <div data-testid={`tool-${event.id}`} data-result-id={result?.id}>
        ToolCard: {event.tool_name}
      </div>
    );
  },
}));

function makeEvent(overrides: Partial<AgentEvent> = {}): AgentEvent {
  return {
    id: "evt-1",
    type: "message",
    role: "assistant",
    content: "",
    timestamp: new Date().toISOString(),
    sequence: 1,
    ...overrides,
  };
}

describe("AgentEventList - message rendering", () => {
  it("renders message events as AgentMessageBubble", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "msg-3",
        type: "message",
        role: "assistant",
        content: "Hello world",
        sequence: 1,
      }),
    ];

    render(<AgentEventList events={events} />);

    const bubble = screen.getByTestId("message-msg-3");
    expect(bubble).toBeInTheDocument();
    expect(bubble).toHaveTextContent("Message: Hello world");
  });
});

describe("AgentEventList - error rendering", () => {
  it("renders error events with error styling", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "err-1",
        type: "error",
        role: "system",
        content: "Something went wrong",
        sequence: 1,
      }),
    ];

    render(<AgentEventList events={events} />);

    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });
});

describe("AgentEventList - status events", () => {
  it("renders status events as null (shown in header, not inline)", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "status-1",
        type: "status",
        role: "system",
        content: "Running",
        sequence: 1,
      }),
    ];

    const { container } = render(<AgentEventList events={events} />);

    expect(screen.queryByText("Running")).not.toBeInTheDocument();
    expect(container.querySelector(".overflow-y-auto")).toBeInTheDocument();
  });
});

describe("AgentEventList - mixed event ordering", () => {
  it("renders a mix of messages, tool calls, and errors in order", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "msg-a",
        type: "message",
        role: "user",
        content: "Do something",
        sequence: 1,
      }),
      makeEvent({
        id: "tc-a",
        type: "tool_call",
        role: "assistant",
        tool_name: "Bash",
        tool_call_id: "call-a",
        sequence: 2,
      }),
      makeEvent({
        id: "tr-a",
        type: "tool_result",
        role: "tool",
        tool_name: "Bash",
        tool_call_id: "call-a",
        tool_output: "done",
        tool_success: true,
        sequence: 3,
      }),
      makeEvent({
        id: "err-a",
        type: "error",
        role: "system",
        content: "Something failed",
        sequence: 4,
      }),
      makeEvent({
        id: "log-a",
        type: "log",
        role: "system",
        content: "debug log",
        sequence: 5,
      }),
      makeEvent({
        id: "metric-a",
        type: "metric",
        role: "system",
        content: "",
        sequence: 6,
      }),
    ];

    render(<AgentEventList events={events} />);

    expect(screen.getByTestId("message-msg-a")).toBeInTheDocument();
    expect(screen.getByTestId("tool-tc-a")).toBeInTheDocument();
    expect(screen.getByText("Something failed")).toBeInTheDocument();
    expect(screen.queryByText("debug log")).not.toBeInTheDocument();
  });
});

describe("AgentEventList - compaction events", () => {
  it("renders compaction events with AgentCompactionCard", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "evt-msg",
        type: "message",
        role: "user",
        content: "Help me fix the auth bug",
      }),
      makeEvent({
        id: "evt-compact",
        type: "compaction",
        role: "system",
        content: "Summary of auth work...",
        sequence: 2,
        compaction_trigger: "manual",
        compaction_focus: "auth",
      }),
      makeEvent({
        id: "evt-msg-2",
        type: "message",
        role: "user",
        content: "Now add rate limiting",
        sequence: 3,
      }),
    ];

    render(<AgentEventList events={events} />);

    expect(screen.getByTestId("message-evt-msg")).toBeInTheDocument();
    expect(screen.getByTestId("compaction-evt-compact")).toBeInTheDocument();
    expect(screen.getByTestId("message-evt-msg-2")).toBeInTheDocument();
  });
});
