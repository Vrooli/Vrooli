/**
 * Tests for AgentFileToolCard component.
 *
 * Verifies rendering of Read, Write, Edit, Glob, and Grep tool call events
 * including:
 * - Correct tool name display per tool type
 * - File path / pattern / grep summary display
 * - Graceful handling of missing input
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AgentFileToolCard } from "./AgentFileToolCard";
import type { AgentEvent } from "../../../../lib/api";

// Mock CodeBlock to avoid shiki/ToastProvider dependency
vi.mock("../../../markdown/components/CodeBlock", () => ({
  CodeBlock: ({ code }: { code: string; language?: string }) => (
    <pre data-testid="code-block">{code}</pre>
  ),
}));

function makeEvent(overrides: Partial<AgentEvent> = {}): AgentEvent {
  return {
    id: "evt-1",
    type: "tool_call",
    role: "assistant",
    content: "",
    timestamp: new Date().toISOString(),
    sequence: 1,
    ...overrides,
  };
}

describe("AgentFileToolCard", () => {
  it("shows 'Read' for Read events", () => {
    const event = makeEvent({
      tool_name: "Read",
      tool_input: JSON.stringify({ file_path: "/src/index.ts" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Read")).toBeInTheDocument();
  });

  it("shows file_path prominently for Read", () => {
    const event = makeEvent({
      tool_name: "Read",
      tool_input: JSON.stringify({ file_path: "/home/user/project/main.go" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("/home/user/project/main.go")).toBeInTheDocument();
  });

  it("shows file_path prominently for Write", () => {
    const event = makeEvent({
      tool_name: "Write",
      tool_input: JSON.stringify({ file_path: "/tmp/output.txt" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Write")).toBeInTheDocument();
    expect(screen.getByText("/tmp/output.txt")).toBeInTheDocument();
  });

  it("shows file_path prominently for Edit", () => {
    const event = makeEvent({
      tool_name: "Edit",
      tool_input: JSON.stringify({ file_path: "/src/app.tsx" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Edit")).toBeInTheDocument();
    expect(screen.getByText("/src/app.tsx")).toBeInTheDocument();
  });

  it("shows pattern prominently for Glob", () => {
    const event = makeEvent({
      tool_name: "Glob",
      tool_input: JSON.stringify({ pattern: "**/*.test.ts" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Glob")).toBeInTheDocument();
    expect(screen.getByText("**/*.test.ts")).toBeInTheDocument();
  });

  it("shows 'pattern in path' format for Grep", () => {
    const event = makeEvent({
      tool_name: "Grep",
      tool_input: JSON.stringify({ pattern: "handleError", path: "/src" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Grep")).toBeInTheDocument();
    expect(screen.getByText("handleError in /src")).toBeInTheDocument();
  });

  it("shows 'pattern in .' when Grep has no path", () => {
    const event = makeEvent({
      tool_name: "Grep",
      tool_input: JSON.stringify({ pattern: "error" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("error in .")).toBeInTheDocument();
  });

  it("shows '? in .' when Grep has no pattern or path", () => {
    const event = makeEvent({
      tool_name: "Grep",
      tool_input: JSON.stringify({}),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("? in .")).toBeInTheDocument();
  });

  it("handles missing tool_input gracefully", () => {
    const event = makeEvent({
      tool_name: "Read",
      tool_input: undefined,
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Read")).toBeInTheDocument();
    expect(screen.getByText("(no input)")).toBeInTheDocument();
  });

  it("handles malformed JSON in tool_input gracefully", () => {
    const event = makeEvent({
      tool_name: "Write",
      tool_input: "not json at all",
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("Write")).toBeInTheDocument();
    expect(screen.getByText("(no input)")).toBeInTheDocument();
  });

  it("shows '(no path)' when Read input has no file_path", () => {
    const event = makeEvent({
      tool_name: "Read",
      tool_input: JSON.stringify({ some_other_key: "value" }),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("(no path)")).toBeInTheDocument();
  });

  it("shows '(no pattern)' when Glob input has no pattern", () => {
    const event = makeEvent({
      tool_name: "Glob",
      tool_input: JSON.stringify({}),
    });

    render(<AgentFileToolCard event={event} />);
    expect(screen.getByText("(no pattern)")).toBeInTheDocument();
  });
});
