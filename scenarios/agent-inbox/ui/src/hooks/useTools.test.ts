/**
 * Tests for useTools hook
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useTools, toolQueryKeys } from "./useTools";
import * as api from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchToolSet: vi.fn(),
  fetchScenarioStatuses: vi.fn(),
  setToolEnabled: vi.fn(),
  setToolApproval: vi.fn(),
  resetToolConfig: vi.fn(),
  refreshTools: vi.fn(),
  syncTools: vi.fn(),
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
        description: "Spawn a coding agent to execute a task",
        category: "agent_lifecycle",
        parameters: {
          type: "object",
          properties: {
            task: {
              type: "string",
              description: "The task to execute",
            },
          },
          required: ["task"],
        },
        metadata: {
          enabled_by_default: true,
          requires_approval: false,
          long_running: true,
          cost_estimate: "high",
          tags: ["coding", "agent"],
        },
      },
      enabled: true,
      source: "",
      requires_approval: false,
    },
    {
      scenario: "agent-manager",
      tool: {
        name: "check_agent_status",
        description: "Check the status of an agent run",
        category: "agent_lifecycle",
        parameters: {
          type: "object",
          properties: {
            run_id: {
              type: "string",
              description: "The run ID to check",
            },
          },
          required: ["run_id"],
        },
        metadata: {
          enabled_by_default: true,
          requires_approval: false,
          cost_estimate: "low",
          tags: ["status"],
        },
      },
      enabled: false,
      source: "global",
      requires_approval: false,
    },
  ],
  categories: [
    {
      id: "agent_lifecycle",
      name: "Agent Lifecycle",
      description: "Tools for managing agent runs",
    },
  ],
  generated_at: "2025-01-01T00:00:00Z",
};

const mockScenarioStatuses: api.ScenarioStatus[] = [
  {
    scenario: "agent-manager",
    available: true,
    last_checked: "2025-01-01T00:00:00Z",
    tool_count: 6,
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
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useTools", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchToolSet).mockResolvedValue(mockToolSet);
    vi.mocked(api.fetchScenarioStatuses).mockResolvedValue(mockScenarioStatuses);
  });

  it("fetches tool set on mount", async () => {
    const { result } = renderHook(() => useTools(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.toolSet).toEqual(mockToolSet);
    expect(api.fetchToolSet).toHaveBeenCalledWith(undefined);
  });

  it("fetches with chatId when provided", async () => {
    const { result } = renderHook(() => useTools({ chatId: "chat-123" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(api.fetchToolSet).toHaveBeenCalledWith("chat-123");
  });

  it("computes enabled tools correctly", async () => {
    const { result } = renderHook(() => useTools(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.enabledTools).toHaveLength(1);
    expect(result.current.enabledTools[0].tool.name).toBe("spawn_coding_agent");
  });

  it("groups tools by scenario", async () => {
    const { result } = renderHook(() => useTools(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.toolsByScenario.size).toBe(1);
    expect(result.current.toolsByScenario.get("agent-manager")).toHaveLength(2);
  });

  it("groups tools by category", async () => {
    const { result } = renderHook(() => useTools(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.toolsByCategory.size).toBe(1);
    expect(result.current.toolsByCategory.get("agent_lifecycle")).toHaveLength(2);
  });

  it("does not fetch when disabled", async () => {
    const { result } = renderHook(() => useTools({ enabled: false }), {
      wrapper: createWrapper(),
    });

    // Give it time to potentially fetch
    await new Promise((r) => setTimeout(r, 50));

    expect(api.fetchToolSet).not.toHaveBeenCalled();
    expect(result.current.toolSet).toBeUndefined();
  });

  it("toggles tool enabled state", async () => {
    vi.mocked(api.setToolEnabled).mockResolvedValue();

    const { result } = renderHook(() => useTools(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await result.current.toggleTool("agent-manager", "spawn_coding_agent", false);

    expect(api.setToolEnabled).toHaveBeenCalledWith({
      chat_id: undefined,
      scenario: "agent-manager",
      tool_name: "spawn_coding_agent",
      enabled: false,
    });
  });

  it("toggles tool with chatId when provided", async () => {
    vi.mocked(api.setToolEnabled).mockResolvedValue();

    const { result } = renderHook(() => useTools({ chatId: "chat-456" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await result.current.toggleTool("agent-manager", "spawn_coding_agent", true);

    expect(api.setToolEnabled).toHaveBeenCalledWith({
      chat_id: "chat-456",
      scenario: "agent-manager",
      tool_name: "spawn_coding_agent",
      enabled: true,
    });
  });

  it("resets tool configuration", async () => {
    vi.mocked(api.resetToolConfig).mockResolvedValue();

    const { result } = renderHook(() => useTools({ chatId: "chat-789" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await result.current.resetTool("agent-manager", "spawn_coding_agent");

    expect(api.resetToolConfig).toHaveBeenCalledWith(
      "agent-manager",
      "spawn_coding_agent",
      "chat-789"
    );
  });

  it("syncs discovered tools", async () => {
    vi.mocked(api.syncTools).mockResolvedValue({
      scenarios_with_tools: 2,
      new_scenarios: ["scenario-to-desktop"],
      removed_scenarios: [],
      total_tools: 26,
    });

    const { result } = renderHook(() => useTools(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await result.current.syncDiscoveredTools();

    expect(api.syncTools).toHaveBeenCalled();
  });
});

describe("toolQueryKeys", () => {
  it("generates correct keys for global tool set", () => {
    expect(toolQueryKeys.toolSet()).toEqual(["tools", "set", "global"]);
  });

  it("generates correct keys for chat-specific tool set", () => {
    expect(toolQueryKeys.toolSet("chat-123")).toEqual(["tools", "set", "chat-123"]);
  });

  it("generates correct keys for scenarios", () => {
    expect(toolQueryKeys.scenarios()).toEqual(["tools", "scenarios"]);
  });
});
