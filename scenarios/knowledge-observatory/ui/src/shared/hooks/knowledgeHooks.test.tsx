import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useHealthStatus, useKnowledgeMetrics, useSearchController } from "./knowledgeHooks";
import type { ApiHealthResponse, HealthResponse, SearchResponse } from "../services/api";
import * as api from "../services/api";
import type { ReactNode } from "react";

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
});
