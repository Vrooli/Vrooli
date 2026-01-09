/**
 * Tests for ChatToolsSelector component
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChatToolsSelector, ToolCountBadge } from "./ChatToolsSelector";
import * as api from "../../lib/api";

// Mock the API module
vi.mock("../../lib/api", () => ({
  fetchToolSet: vi.fn(),
  fetchScenarioStatuses: vi.fn(),
  setToolEnabled: vi.fn(),
  resetToolConfig: vi.fn(),
  refreshTools: vi.fn(),
}));

const mockToolSet: api.ToolSet = {
  scenarios: [
    {
      name: "agent-manager",
      version: "1.0.0",
      description: "Manages coding agents",
    },
  ],
  tools: [
    {
      scenario: "agent-manager",
      tool: {
        name: "spawn_coding_agent",
        description: "Spawn a coding agent",
        parameters: { type: "object", properties: {} },
        metadata: { enabled_by_default: true, requires_approval: false },
      },
      enabled: true,
      source: "",
      requires_approval: false,
    },
    {
      scenario: "agent-manager",
      tool: {
        name: "check_status",
        description: "Check agent status",
        parameters: { type: "object", properties: {} },
        metadata: { enabled_by_default: true, requires_approval: false },
      },
      enabled: false,
      source: "chat",
      requires_approval: false,
    },
  ],
  categories: [],
  generated_at: "2025-01-01T00:00:00Z",
};

const mockScenarioStatuses: api.ScenarioStatus[] = [
  {
    scenario: "agent-manager",
    available: true,
    last_checked: "2025-01-01T00:00:00Z",
    tool_count: 2,
  },
];

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("ChatToolsSelector", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchToolSet).mockResolvedValue(mockToolSet);
    vi.mocked(api.fetchScenarioStatuses).mockResolvedValue(mockScenarioStatuses);
  });

  it("renders trigger button with tool count", async () => {
    render(<ChatToolsSelector chatId="chat-123" />, { wrapper: createWrapper() });

    // Wait for data to load and count to appear
    await waitFor(() => {
      expect(screen.getByText("1/2")).toBeInTheDocument();
    });

    expect(screen.getByTestId("chat-tools-trigger")).toBeInTheDocument();
  });

  it("opens popover when trigger is clicked", async () => {
    render(<ChatToolsSelector chatId="chat-123" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-trigger")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("chat-tools-trigger"));

    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-popover")).toBeInTheDocument();
    });

    expect(screen.getByText("Chat Tools")).toBeInTheDocument();
  });

  it("closes popover when close button is clicked", async () => {
    render(<ChatToolsSelector chatId="chat-123" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-trigger")).toBeInTheDocument();
    });

    // Open
    fireEvent.click(screen.getByTestId("chat-tools-trigger"));
    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-popover")).toBeInTheDocument();
    });

    // Close
    fireEvent.click(screen.getByTestId("chat-tools-close"));
    await waitFor(() => {
      expect(screen.queryByTestId("chat-tools-popover")).not.toBeInTheDocument();
    });
  });

  it("closes popover on Escape key", async () => {
    render(<ChatToolsSelector chatId="chat-123" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-trigger")).toBeInTheDocument();
    });

    // Open
    fireEvent.click(screen.getByTestId("chat-tools-trigger"));
    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-popover")).toBeInTheDocument();
    });

    // Press Escape
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() => {
      expect(screen.queryByTestId("chat-tools-popover")).not.toBeInTheDocument();
    });
  });

  it("renders ToolConfiguration inside popover", async () => {
    render(<ChatToolsSelector chatId="chat-123" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId("chat-tools-trigger")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("chat-tools-trigger"));

    await waitFor(() => {
      expect(screen.getByTestId("tool-configuration")).toBeInTheDocument();
    });
  });

  it("shows override indicator when tools have overrides", async () => {
    // Override mock to have an enabled tool with chat-level override
    const overrideToolSet = {
      ...mockToolSet,
      tools: [
        {
          ...mockToolSet.tools[0],
          source: "chat" as const, // This enabled tool has a chat override
        },
      ],
    };
    vi.mocked(api.fetchToolSet).mockResolvedValue(overrideToolSet);

    render(<ChatToolsSelector chatId="chat-123" />, { wrapper: createWrapper() });

    // Wait for data to load
    await waitFor(() => {
      expect(screen.getByText("1/1")).toBeInTheDocument();
    });

    // The trigger should have indigo color when overrides exist
    const trigger = screen.getByTestId("chat-tools-trigger");
    expect(trigger.className).toContain("text-indigo-400");
  });

  it("fetches tools with correct chatId", async () => {
    render(<ChatToolsSelector chatId="my-chat-id" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(api.fetchToolSet).toHaveBeenCalledWith("my-chat-id");
    });
  });
});

describe("ToolCountBadge", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchToolSet).mockResolvedValue(mockToolSet);
    vi.mocked(api.fetchScenarioStatuses).mockResolvedValue(mockScenarioStatuses);
  });

  it("renders enabled tool count", async () => {
    render(<ToolCountBadge chatId="chat-123" />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId("tool-count-badge")).toBeInTheDocument();
    });

    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("returns null when loading", () => {
    // Mock slow response
    vi.mocked(api.fetchToolSet).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    const { container } = render(<ToolCountBadge chatId="chat-123" />, {
      wrapper: createWrapper(),
    });

    expect(container.firstChild).toBeNull();
  });

  it("returns null when no tools", async () => {
    vi.mocked(api.fetchToolSet).mockResolvedValue({
      ...mockToolSet,
      tools: [],
    });

    render(<ToolCountBadge chatId="chat-123" />, {
      wrapper: createWrapper(),
    });

    // Wait for fetch to complete and component to update
    await waitFor(() => {
      expect(api.fetchToolSet).toHaveBeenCalled();
    });

    // Badge should not appear when there are no tools
    await waitFor(() => {
      expect(screen.queryByTestId("tool-count-badge")).not.toBeInTheDocument();
    });
  });
});
