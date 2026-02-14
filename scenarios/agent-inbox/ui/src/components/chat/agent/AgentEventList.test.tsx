/**
 * Tests for AgentEventList component.
 *
 * Verifies event grouping, filtering, and rendering including:
 * - tool_call + tool_result grouping by tool_call_id
 * - Fallback name+proximity matching when tool_call_id is absent
 * - Metric and log events are filtered out
 * - Message events render as AgentMessageBubble
 * - Error events render with error styling
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AgentEventList } from "./AgentEventList";
import type { AgentEvent } from "../../../lib/api";

// Mock child components to avoid rendering complexities (CodeBlock with shiki,
// MarkdownRenderer, etc.) and focus on grouping/filtering logic.
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

vi.mock("./tools", () => ({
  getToolComponent: (_toolName?: string, _runnerType?: string) => {
    // Return a simple component that renders with a data-testid
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

describe("AgentEventList", () => {
  it("renders waiting message when events array is empty", () => {
    render(<AgentEventList events={[]} />);
    expect(screen.getByText("Waiting for agent response...")).toBeInTheDocument();
  });

  describe("tool_call + tool_result grouping by tool_call_id", () => {
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

      // The tool call should be rendered with the result attached
      const toolCard = screen.getByTestId("tool-call-1");
      expect(toolCard).toBeInTheDocument();
      expect(toolCard).toHaveAttribute("data-result-id", "result-1");

      // The result should NOT be rendered as a separate standalone card
      expect(screen.queryByTestId("tool-result-1")).not.toBeInTheDocument();
    });
  });

  describe("fallback name+proximity matching", () => {
    it("falls back to name+proximity matching when tool_call_id is absent", () => {
      const events: AgentEvent[] = [
        makeEvent({
          id: "call-2",
          type: "tool_call",
          role: "assistant",
          tool_name: "Read",
          // No tool_call_id
          sequence: 1,
        }),
        makeEvent({
          id: "result-2",
          type: "tool_result",
          role: "tool",
          tool_name: "Read",
          // No tool_call_id
          tool_output: "file content",
          sequence: 2,
        }),
      ];

      render(<AgentEventList events={events} />);

      // The tool call should be rendered with the result attached via fallback matching
      const toolCard = screen.getByTestId("tool-call-2");
      expect(toolCard).toBeInTheDocument();
      expect(toolCard).toHaveAttribute("data-result-id", "result-2");

      // The result should NOT appear standalone
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
        // Insert 11 filler events to exceed proximity window
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

      // The tool call should NOT have the distant result attached
      const toolCard = screen.getByTestId("tool-call-far");
      expect(toolCard).toBeInTheDocument();
      // data-result-id should not be set (result was too far away)
      expect(toolCard.getAttribute("data-result-id")).toBeFalsy();
    });
  });

  describe("event filtering", () => {
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
        // Also include a message so the list isn't empty
        makeEvent({
          id: "msg-1",
          type: "message",
          role: "assistant",
          content: "Hello",
          sequence: 2,
        }),
      ];

      render(<AgentEventList events={events} />);

      // Metric event should not be rendered
      expect(screen.queryByTestId("raw-metric-1")).not.toBeInTheDocument();
      // But the message should be
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

  describe("message rendering", () => {
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

  describe("error rendering", () => {
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

  describe("status events", () => {
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

      // Status events return null, so the content area should not contain status text
      // but the container should exist since events.length > 0
      expect(screen.queryByText("Running")).not.toBeInTheDocument();
      expect(container.querySelector(".overflow-y-auto")).toBeInTheDocument();
    });
  });

  describe("mixed event ordering", () => {
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

      // Message rendered
      expect(screen.getByTestId("message-msg-a")).toBeInTheDocument();
      // Tool call rendered (with result grouped)
      expect(screen.getByTestId("tool-tc-a")).toBeInTheDocument();
      // Error rendered
      expect(screen.getByText("Something failed")).toBeInTheDocument();
      // Log and metric NOT rendered
      expect(screen.queryByText("debug log")).not.toBeInTheDocument();
    });
  });
});
