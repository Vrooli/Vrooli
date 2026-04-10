/**
 * Tests for AgentEventList - Event grouping, filtering, and basic rendering
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

describe("AgentEventList - basic rendering", () => {
  it("renders waiting message when events array is empty", () => {
    render(<AgentEventList events={[]} />);
    expect(screen.getByText("Waiting for agent response...")).toBeInTheDocument();
  });

  it("applies compact spacing when compact view mode is selected", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "msg-compact",
        type: "message",
        role: "assistant",
        content: "compact",
        sequence: 1,
      }),
    ];

    render(<AgentEventList events={events} viewMode="compact" />);

    expect(screen.getByTestId("agent-event-list")).toHaveClass("space-y-2");
  });
});

describe("AgentEventList - tool_call + tool_result grouping by tool_call_id", () => {
  it("groups tool_call and tool_result by tool_call_id", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "call-1",
        type: "tool_call",
        role: "assistant",
        tool_name: "Bash",
        tool_call_id: "tc-123",
        sequence: 1,
      }),
      makeEvent({
        id: "result-1",
        type: "tool_result",
        role: "tool",
        tool_name: "Bash",
        tool_call_id: "tc-123",
        tool_output: "hello",
        sequence: 2,
      }),
    ];

    render(<AgentEventList events={events} />);

    const toolCard = screen.getByTestId("tool-call-1");
    expect(toolCard).toBeInTheDocument();
    expect(toolCard).toHaveAttribute("data-result-id", "result-1");

    expect(screen.queryByTestId("tool-result-1")).not.toBeInTheDocument();
  });
});

describe("AgentEventList - fallback name+proximity matching", () => {
  it("falls back to name+proximity matching when tool_call_id is absent", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "call-2",
        type: "tool_call",
        role: "assistant",
        tool_name: "Read",
        sequence: 1,
      }),
      makeEvent({
        id: "result-2",
        type: "tool_result",
        role: "tool",
        tool_name: "Read",
        tool_output: "file content",
        sequence: 2,
      }),
    ];

    render(<AgentEventList events={events} />);

    const toolCard = screen.getByTestId("tool-call-2");
    expect(toolCard).toBeInTheDocument();
    expect(toolCard).toHaveAttribute("data-result-id", "result-2");

    expect(screen.queryByTestId("tool-result-2")).not.toBeInTheDocument();
  });

  it("does not match results beyond 10-event proximity window", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "call-far",
        type: "tool_call",
        role: "assistant",
        tool_name: "Bash",
        sequence: 1,
      }),
      ...Array.from({ length: 11 }, (_, i) =>
        makeEvent({
          id: `filler-${i}`,
          type: "message",
          role: "assistant",
          content: `filler ${i}`,
          sequence: i + 2,
        })
      ),
      makeEvent({
        id: "result-far",
        type: "tool_result",
        role: "tool",
        tool_name: "Bash",
        tool_output: "output",
        sequence: 13,
      }),
    ];

    render(<AgentEventList events={events} />);

    const toolCard = screen.getByTestId("tool-call-far");
    expect(toolCard).toBeInTheDocument();
    expect(toolCard.getAttribute("data-result-id")).toBeFalsy();
  });
});

describe("AgentEventList - event filtering", () => {
  it("does NOT render metric events in the list", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "metric-1",
        type: "metric",
        role: "system",
        content: "",
        raw_data: JSON.stringify({ name: "tokens", value: 100 }),
        sequence: 1,
      }),
      makeEvent({
        id: "msg-1",
        type: "message",
        role: "assistant",
        content: "Hello",
        sequence: 2,
      }),
    ];

    render(<AgentEventList events={events} />);

    expect(screen.queryByTestId("raw-metric-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("message-msg-1")).toBeInTheDocument();
  });

  it("does NOT render log events in the list", () => {
    const events: AgentEvent[] = [
      makeEvent({
        id: "log-1",
        type: "log",
        role: "system",
        content: "Debug info",
        sequence: 1,
      }),
      makeEvent({
        id: "msg-2",
        type: "message",
        role: "assistant",
        content: "Response",
        sequence: 2,
      }),
    ];

    render(<AgentEventList events={events} />);

    expect(screen.queryByTestId("raw-log-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("message-msg-2")).toBeInTheDocument();
  });
});
