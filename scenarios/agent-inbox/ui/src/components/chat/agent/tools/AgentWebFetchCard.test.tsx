/**
 * Tests for AgentWebFetchCard component.
 *
 * Verifies rendering of WebFetch and WebSearch tool call events including:
 * - URL display for WebFetch
 * - Query display for WebSearch
 * - Clickable link rendering when expanded
 * - Graceful handling of missing input
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AgentWebFetchCard } from "./AgentWebFetchCard";
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

describe("AgentWebFetchCard", () => {
  describe("WebFetch", () => {
    it("shows URL prominently for WebFetch", () => {
      const event = makeEvent({
        tool_name: "WebFetch",
        tool_input: JSON.stringify({
          url: "https://example.com/api/data",
          prompt: "Extract the main content",
        }),
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("WebFetch")).toBeInTheDocument();
      // The URL appears as the summary in the collapsed header
      expect(screen.getByText("https://example.com/api/data")).toBeInTheDocument();
    });

    it("renders URL as a clickable link when expanded", async () => {
      const user = userEvent.setup();
      const event = makeEvent({
        tool_name: "WebFetch",
        tool_input: JSON.stringify({
          url: "https://docs.example.com/guide",
          prompt: "Summarize",
        }),
      });

      render(<AgentWebFetchCard event={event} />);

      // Click to expand
      await user.click(screen.getByRole("button"));

      // The link should be rendered inside the expanded section
      const link = screen.getByRole("link", { name: "https://docs.example.com/guide" });
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute("href", "https://docs.example.com/guide");
      expect(link).toHaveAttribute("target", "_blank");
      expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });

    it("shows '(no URL)' when WebFetch has no url", () => {
      const event = makeEvent({
        tool_name: "WebFetch",
        tool_input: JSON.stringify({ prompt: "Get something" }),
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("(no URL)")).toBeInTheDocument();
    });
  });

  describe("WebSearch", () => {
    it("shows query prominently for WebSearch", () => {
      const event = makeEvent({
        tool_name: "WebSearch",
        tool_input: JSON.stringify({ query: "vitest testing best practices" }),
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("WebSearch")).toBeInTheDocument();
      expect(screen.getByText("vitest testing best practices")).toBeInTheDocument();
    });

    it("shows '(no query)' when WebSearch has no query", () => {
      const event = makeEvent({
        tool_name: "WebSearch",
        tool_input: JSON.stringify({}),
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("(no query)")).toBeInTheDocument();
    });
  });

  describe("missing/malformed input", () => {
    it("handles missing tool_input gracefully for WebFetch", () => {
      const event = makeEvent({
        tool_name: "WebFetch",
        tool_input: undefined,
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("WebFetch")).toBeInTheDocument();
      expect(screen.getByText("(no URL)")).toBeInTheDocument();
    });

    it("handles missing tool_input gracefully for WebSearch", () => {
      const event = makeEvent({
        tool_name: "WebSearch",
        tool_input: undefined,
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("WebSearch")).toBeInTheDocument();
      expect(screen.getByText("(no query)")).toBeInTheDocument();
    });

    it("handles malformed JSON in tool_input gracefully", () => {
      const event = makeEvent({
        tool_name: "WebFetch",
        tool_input: "{{invalid json}}",
      });

      render(<AgentWebFetchCard event={event} />);
      expect(screen.getByText("WebFetch")).toBeInTheDocument();
      expect(screen.getByText("(no URL)")).toBeInTheDocument();
    });
  });
});
