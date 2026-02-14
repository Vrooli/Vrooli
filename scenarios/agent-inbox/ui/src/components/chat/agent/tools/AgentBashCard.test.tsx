/**
 * Tests for AgentBashCard component.
 *
 * Verifies rendering of Bash tool call events including:
 * - Tool name display
 * - Description/command summary in collapsed view
 * - Expand/collapse behavior
 * - Success/failure badges
 * - Graceful handling of missing or malformed input
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AgentBashCard } from "./AgentBashCard";
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

describe("AgentBashCard", () => {
  it("renders 'Bash' as tool name", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({ command: "ls -la" }),
    });

    render(<AgentBashCard event={event} />);
    expect(screen.getByText("Bash")).toBeInTheDocument();
  });

  it("shows description prominently in collapsed view", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({
        command: "git status",
        description: "Check working tree status",
      }),
    });

    render(<AgentBashCard event={event} />);
    expect(screen.getByText("Check working tree status")).toBeInTheDocument();
  });

  it("falls back to command when no description is provided", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({ command: "npm install" }),
    });

    render(<AgentBashCard event={event} />);
    expect(screen.getByText("npm install")).toBeInTheDocument();
  });

  it("shows '(no command)' when neither description nor command is present", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({}),
    });

    render(<AgentBashCard event={event} />);
    expect(screen.getByText("(no command)")).toBeInTheDocument();
  });

  it("clicking expands to show command and output", async () => {
    const user = userEvent.setup();
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({
        command: "echo hello",
        description: "Print hello",
      }),
    });
    const result = makeEvent({
      id: "evt-2",
      type: "tool_result",
      tool_name: "Bash",
      tool_output: "hello",
      tool_success: true,
    });

    render(<AgentBashCard event={event} result={result} />);

    // Before clicking, the output section should not be visible
    expect(screen.queryByText("Command")).not.toBeInTheDocument();
    expect(screen.queryByText("Output")).not.toBeInTheDocument();

    // Click the header to expand
    await user.click(screen.getByRole("button"));

    // After expanding, command and output labels should be visible
    expect(screen.getByText("Command")).toBeInTheDocument();
    expect(screen.getByText("Output")).toBeInTheDocument();
    // The actual command text and output should appear in the CodeBlock
    expect(screen.getByText("echo hello")).toBeInTheDocument();
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("shows success badge when result.tool_success is true", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({ command: "ls" }),
    });
    const result = makeEvent({
      id: "evt-2",
      type: "tool_result",
      tool_success: true,
    });

    render(<AgentBashCard event={event} result={result} />);
    expect(screen.getByText("Success")).toBeInTheDocument();
  });

  it("shows failure badge when result.tool_success is false", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: JSON.stringify({ command: "false" }),
    });
    const result = makeEvent({
      id: "evt-2",
      type: "tool_result",
      tool_success: false,
    });

    render(<AgentBashCard event={event} result={result} />);
    expect(screen.getByText("Failed")).toBeInTheDocument();
  });

  it("handles missing tool_input gracefully", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: undefined,
    });

    render(<AgentBashCard event={event} />);
    expect(screen.getByText("(no command)")).toBeInTheDocument();
    expect(screen.getByText("Bash")).toBeInTheDocument();
  });

  it("handles malformed JSON in tool_input gracefully", () => {
    const event = makeEvent({
      tool_name: "Bash",
      tool_input: "not valid json {{{",
    });

    render(<AgentBashCard event={event} />);
    // parseToolInput returns null, so description and command are both undefined
    expect(screen.getByText("(no command)")).toBeInTheDocument();
    expect(screen.getByText("Bash")).toBeInTheDocument();
  });
});
