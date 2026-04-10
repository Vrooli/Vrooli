import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useHealthStatus,
  useKnowledgeGraphController,
  useKnowledgeMetrics,
  useSearchController,
} from "./knowledgeHooks";
import type { ApiHealthResponse, GraphResponse, HealthResponse, SearchResponse } from "../services/api";
import * as api from "../services/api";
import type { FormEvent, ReactNode } from "react";

vi.mock("../services/api");

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe("knowledgeHooks", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it("useHealthStatus returns a view model", async () => {
    const fetchHealthMock = vi.mocked(api.fetchHealth);
    const healthPayload: ApiHealthResponse = {
      status: "ok",
      service: "knowledge-observatory",
      timestamp: "2026-01-25T12:00:00Z",
    };
    fetchHealthMock.mockResolvedValue(healthPayload);

    const { result } = renderHook(() => useHealthStatus(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.hasData).toBe(true);
    });

    expect(result.current.viewModel.status).toBe("ok");
    expect(result.current.viewModel.statusLabel).toBe("Online");
  });

  it("useSearchController runs queries and maps results", async () => {
    const searchMock = vi.mocked(api.searchKnowledge);
    const payload: SearchResponse = {
      results: [
        {
          id: "r-1",
          score: 0.9,
          content: "Result content",
          metadata: { source: "test" },
        },
      ],
      query: "semantic query",
      took_ms: 5,
    };
    searchMock.mockResolvedValue(payload);

    const { result } = renderHook(() => useSearchController(), { wrapper: createWrapper() });

    act(() => {
      result.current.runSearch("semantic query");
    });

    await waitFor(() => {
      expect(result.current.viewModel.totalResults).toBe(1);
    });

    expect(searchMock).toHaveBeenCalledWith({ query: "semantic query", limit: 10 });
    expect(result.current.viewModel.results[0]?.id).toBe("r-1");
  });

  it("useKnowledgeMetrics exposes view model values", async () => {
    const metricsMock = vi.mocked(api.fetchKnowledgeHealth);
    const payload: HealthResponse = {
      total_entries: 42,
      collections: [],
      overall_health: "steady",
      overall_metrics: { coherence: 0.65 },
      timestamp: "2026-01-25T10:00:00Z",
    };
    metricsMock.mockResolvedValue(payload);

    const { result } = renderHook(() => useKnowledgeMetrics(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.hasData).toBe(true);
    });

    expect(result.current.viewModel.overallHealth).toBe("steady");
    expect(result.current.viewModel.hasMetrics).toBe(true);
    expect(result.current.viewModel.metricCards[0]?.label).toBe("Coherence");
    expect(result.current.viewModel.totalEntriesLabel).toBe((42).toLocaleString());
  });

  it("useKnowledgeGraphController submits graph request and maps response", async () => {
    const graphMock = vi.mocked(api.fetchKnowledgeGraph);
    const payload: GraphResponse = {
      center: "semantic drift",
      nodes: [{ id: "center", label: "semantic drift", score: 1, metadata: { type: "center" } }],
      edges: [{ source: "center", target: "node-1", weight: 0.87, relationship: "semantic_similarity" }],
      took_ms: 16,
    };
    graphMock.mockResolvedValue(payload);

    const { result } = renderHook(() => useKnowledgeGraphController(), { wrapper: createWrapper() });

    act(() => {
      result.current.setCenterConcept("semantic drift");
      result.current.setLimitInput("20");
    });

    act(() => {
      const submitEvent = {
        preventDefault: vi.fn(),
      } as unknown as FormEvent<HTMLFormElement>;
      result.current.submit(submitEvent);
    });

    await waitFor(() => {
      expect(result.current.hasData).toBe(true);
    });

    expect(graphMock).toHaveBeenCalledWith({
      center_concept: "semantic drift",
      collection: undefined,
      namespaces: [],
      tags: [],
      visibility: ["shared", "global"],
      depth: 1,
      limit: 20,
      threshold: 0.35,
    });
    expect(result.current.viewModel.nodeCount).toBe(1);
    expect(result.current.viewModel.edgeCount).toBe(1);

    await expect(result.current.queryGraph("semantic drift")).resolves.toEqual(payload);
  });
});
