/**
 * Integration tests for useTools hook - toolsByScenario derivation and concurrent mutations
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useTools } from "./useTools";
import * as api from "../lib/api";
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

describe("useTools - toolsByScenario derivation", () => {
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

  it("toolsByScenario should correctly group tools", async () => {
    const initialToolSet = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", false),
      createMockTool("scenario-b", "tool1", true),
    ]);

    vi.mocked(api.fetchToolSet).mockResolvedValue(initialToolSet);

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.toolsByScenario.size).toBe(2);
    expect(result.current.toolsByScenario.get("scenario-a")).toHaveLength(2);
    expect(result.current.toolsByScenario.get("scenario-b")).toHaveLength(1);
  });

  it("toolsByScenario should update after optimistic update", async () => {
    let serverState = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ]);

    vi.mocked(api.fetchToolSet).mockImplementation(() => Promise.resolve(serverState));
    vi.mocked(api.setToolEnabled).mockImplementation((config) => {
      serverState = {
        ...serverState,
        tools: serverState.tools.map((t) =>
          t.scenario === config.scenario && t.tool.name === config.tool_name
            ? { ...t, enabled: config.enabled }
            : t
        ),
      };
      return Promise.resolve();
    });

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    const scenarioAToolsBefore = result.current.toolsByScenario.get("scenario-a");
    expect(scenarioAToolsBefore).toBeDefined();
    expect(scenarioAToolsBefore?.[0]?.enabled).toBe(true);
    expect(scenarioAToolsBefore?.[1]?.enabled).toBe(true);

    await act(async () => {
      await result.current.toggleTool("scenario-a", "tool1", false);
    });

    await act(async () => {
      await flushPromises();
    });

    await waitFor(() => {
      const scenarioAToolsAfter = result.current.toolsByScenario.get("scenario-a");
      expect(scenarioAToolsAfter).toBeDefined();
      expect(scenarioAToolsAfter?.[0]?.enabled).toBe(false);
      expect(scenarioAToolsAfter?.[1]?.enabled).toBe(true);
    });
  });
});

describe("useTools - concurrent mutations", () => {
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

  it("rapid toggles should all be reflected in final state", async () => {
    let serverState = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ]);

    vi.mocked(api.fetchToolSet).mockImplementation(() => Promise.resolve(serverState));
    vi.mocked(api.setToolEnabled).mockImplementation((config) => {
      serverState = {
        ...serverState,
        tools: serverState.tools.map((t) =>
          t.scenario === config.scenario && t.tool.name === config.tool_name
            ? { ...t, enabled: config.enabled }
            : t
        ),
      };
      return Promise.resolve();
    });

    const { result } = renderHook(() => useTools(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      const p1 = result.current.toggleTool("scenario-a", "tool1", false);
      const p2 = result.current.toggleTool("scenario-a", "tool2", false);
      const p3 = result.current.toggleTool("scenario-a", "tool3", false);
      await Promise.all([p1, p2, p3]);
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
  });
});
