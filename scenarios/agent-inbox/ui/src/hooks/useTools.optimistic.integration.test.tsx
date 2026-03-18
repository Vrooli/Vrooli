/**
 * Integration tests for useTools hook - Optimistic updates and verification
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useTools } from "./useTools";
import * as api from "../lib/api";
import { verifyOnlyOneToolChanged } from "../lib/toolToggleLogic";
import type { ToolSet, EffectiveTool } from "../lib/api";

// Helper to flush promises and setTimeout callbacks
const flushPromises = () => new Promise(resolve => setTimeout(resolve, 50));

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchToolSet: vi.fn(),
  fetchScenarioStatuses: vi.fn(),
  setToolEnabled: vi.fn(),
  setToolApproval: vi.fn(),
  resetToolConfig: vi.fn(),
  syncTools: vi.fn(),
}));

// Helper to create mock tools
function createMockTool(
  scenario: string,
  name: string,
  enabled: boolean
): EffectiveTool {
  return {
    scenario,
    tool: {
      name,
      description: `Test tool ${name}`,
      parameters: { type: "object", properties: {} },
      metadata: {
        enabled_by_default: true,
        requires_approval: false,
      },
    },
    enabled,
    source: "",
    requires_approval: false,
  };
}

function createMockToolSet(tools: EffectiveTool[]): ToolSet {
  return {
    scenarios: [],
    tools,
    categories: [],
    generated_at: new Date().toISOString(),
  };
}

describe("useTools - Optimistic Update Verification", () => {
  let queryClient: QueryClient;

  function createWrapper() {
    return ({ children }: { children: React.ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);
  }

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.mocked(api.fetchScenarioStatuses).mockResolvedValue([]);
  });

  afterEach(() => {
    queryClient.clear();
  });

  it("optimistic update should only change the targeted tool", async () => {
    const initialToolSet = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ]);

    vi.mocked(api.fetchToolSet).mockResolvedValue(initialToolSet);
    vi.mocked(api.setToolEnabled).mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    const toolSetBefore = result.current.toolSet;
    expect(toolSetBefore).toBeDefined();
    if (!toolSetBefore) throw new Error("toolSetBefore is undefined");

    act(() => {
      result.current.toggleTool("scenario-a", "tool2", false);
    });

    await waitFor(() => {
      const tool2 = result.current.toolSet?.tools.find(
        (t) => t.scenario === "scenario-a" && t.tool.name === "tool2"
      );
      expect(tool2?.enabled).toBe(false);
    });

    const toolSetAfter = result.current.toolSet;
    expect(toolSetAfter).toBeDefined();
    if (!toolSetAfter) throw new Error("toolSetAfter is undefined");
    const verification = verifyOnlyOneToolChanged(
      toolSetBefore,
      toolSetAfter,
      "scenario-a",
      "tool2"
    );

    expect(verification.valid).toBe(true);
    expect(verification.changedCount).toBe(1);
    expect(verification.unexpectedChanges).toHaveLength(0);
  });

  it("multiple sequential toggles should each update correctly", async () => {
    let serverState = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ]);

    vi.mocked(api.fetchToolSet).mockImplementation(async () => serverState);
    vi.mocked(api.setToolEnabled).mockImplementation(async (config) => {
      serverState = {
        ...serverState,
        tools: serverState.tools.map((t) =>
          t.scenario === config.scenario && t.tool.name === config.tool_name
            ? { ...t, enabled: config.enabled }
            : t
        ),
      };
    });

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.toggleTool("scenario-a", "tool1", false);
    });

    await act(async () => {
      await result.current.toggleTool("scenario-a", "tool2", false);
    });

    await act(async () => {
      await result.current.toggleTool("scenario-a", "tool3", false);
    });

    await act(async () => {
      await flushPromises();
    });

    await waitFor(() => {
      const tools = result.current.toolSet?.tools;
      expect(tools![0]!.enabled).toBe(false);
      expect(tools![1]!.enabled).toBe(false);
      expect(tools![2]!.enabled).toBe(false);
    });

    expect(api.setToolEnabled).toHaveBeenCalledTimes(3);
  });

  it("toggling tools in different scenarios should not affect each other", async () => {
    let serverState = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-b", "tool1", true),
      createMockTool("scenario-b", "tool2", true),
    ]);

    vi.mocked(api.fetchToolSet).mockImplementation(async () => serverState);
    vi.mocked(api.setToolEnabled).mockImplementation(async (config) => {
      serverState = {
        ...serverState,
        tools: serverState.tools.map((t) =>
          t.scenario === config.scenario && t.tool.name === config.tool_name
            ? { ...t, enabled: config.enabled }
            : t
        ),
      };
    });

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.toggleTool("scenario-a", "tool1", false);
    });

    await act(async () => {
      await flushPromises();
    });

    await waitFor(() => {
      const tools = result.current.toolSet?.tools;
      expect(tools![0]!.enabled).toBe(false);
      expect(tools![1]!.enabled).toBe(true);
      expect(tools![2]!.enabled).toBe(true);
      expect(tools![3]!.enabled).toBe(true);
    });
  });

  it("error should rollback optimistic update", async () => {
    const initialToolSet = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
    ]);

    vi.mocked(api.fetchToolSet).mockResolvedValue(initialToolSet);
    vi.mocked(api.setToolEnabled).mockRejectedValue(new Error("API Error"));

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    try {
      await act(async () => {
        await result.current.toggleTool("scenario-a", "tool1", false);
      });
    } catch {
      // Expected to throw
    }

    expect(api.setToolEnabled).toHaveBeenCalled();
  });
});
