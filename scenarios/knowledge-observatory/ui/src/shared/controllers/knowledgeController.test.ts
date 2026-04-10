import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  buildGraphViewModel,
  buildHealthViewModel,
  buildMetricsViewModel,
  buildSearchViewModel,
  formatTimestamp,
  loadKnowledgeGraph,
  loadHealth,
  loadKnowledgeMetrics,
  runGraphQuery,
  runSearchQuery,
} from "./knowledgeController";
import type { ApiHealthResponse, GraphResponse, HealthResponse, SearchResponse } from "../services/api";
import * as api from "../services/api";

vi.mock("../services/api");

describe("knowledgeController", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it("formatTimestamp guards invalid dates", () => {
    expect(formatTimestamp("not-a-date")).toBe("Unknown");
    expect(formatTimestamp()).toBe("Unknown");
  });

  it("buildHealthViewModel provides safe defaults", () => {
    const viewModel = buildHealthViewModel({
      data: null,
      isLoading: false,
      hasError: true,
    });

    expect(viewModel.status).toBe("Unknown");
    expect(viewModel.service).toBe("Unknown");
    expect(viewModel.statusLabel).toBe("Offline");
    expect(viewModel.statusPulse).toBe(false);
  });

  it("buildSearchViewModel normalizes empty payloads", () => {
    const data = {
      results: [{ id: "", score: Number.NaN, content: "", metadata: null }],
      query: "",
      took_ms: Number.NaN,
    } as unknown as SearchResponse;

    const viewModel = buildSearchViewModel({
      data,
      fallbackQuery: "fallback query",
      error: null,
    });

    expect(viewModel.displayQuery).toBe("fallback query");
    expect(viewModel.totalResults).toBe(1);
    expect(viewModel.results[0]?.id).toBe("result-1");
    expect(viewModel.results[0]?.scoreLabel).toBe("N/A");
    expect(viewModel.results[0]?.content).toBe("No content available");
    expect(viewModel.results[0]?.metadata).toEqual({});
    expect(viewModel.results[0]?.hasMetadata).toBe(false);
    expect(viewModel.tookMsLabel).toBe("?ms");
    expect(viewModel.hasResults).toBe(true);
  });

  it("buildMetricsViewModel handles missing data", () => {
    const viewModel = buildMetricsViewModel(null);
    expect(viewModel.totalEntriesLabel).toBe("Unknown");
    expect(viewModel.collections).toEqual([]);
    expect(viewModel.hasMetrics).toBe(false);
    expect(viewModel.metricCards).toEqual([]);
  });

  it("buildMetricsViewModel maps metric cards and collections", () => {
    const viewModel = buildMetricsViewModel({
      total_entries: 42,
      collections: [
        {
          name: "alpha",
          size: 12,
          metrics: { coherence: 0.65, redundancy: 0.3 },
        },
      ],
      overall_health: "steady",
      overall_metrics: { coherence: 0.7, redundancy: 0.5 },
      timestamp: "2026-01-25T10:00:00Z",
    });

    expect(viewModel.hasMetrics).toBe(true);
    expect(viewModel.metricCards.map((card) => card.label)).toEqual(["Coherence", "Redundancy"]);
    expect(viewModel.metricCards[0]?.percentageLabel).toBe("70.0%");
    expect(viewModel.metricCards[1]?.tone).toBe("poor");
    expect(viewModel.collections[0]?.name).toBe("alpha");
    expect(viewModel.collections[0]?.sizeLabel).toBe("12 vectors");
    expect(viewModel.collections[0]?.metrics).toEqual([
      { label: "Coherence", percentageLabel: "65%" },
      { label: "Redundancy", percentageLabel: "30%" },
    ]);
  });

  it("runSearchQuery validates and calls the service", async () => {
    const searchMock = vi.mocked(api.searchKnowledge);
    const payload: SearchResponse = {
      results: [],
      query: "semantic query",
      took_ms: 5,
    };
    searchMock.mockResolvedValue(payload);

    await expect(runSearchQuery("semantic query")).resolves.toEqual(payload);
    expect(searchMock).toHaveBeenCalledWith({ query: "semantic query", limit: 10 });
  });

  it("runSearchQuery rejects empty queries", async () => {
    await expect(runSearchQuery("   ")).rejects.toThrow("Search query is missing.");
  });

  it("buildGraphViewModel maps nodes and edges", () => {
    const viewModel = buildGraphViewModel({
      data: {
        center: "semantic drift",
        nodes: [{ id: "center", label: "semantic drift", score: 1, metadata: { type: "center" } }],
        edges: [
          {
            source: "center",
            target: "node-1",
            weight: 0.82,
            relationship: "semantic_similarity",
          },
        ],
        took_ms: 37,
      },
      fallbackCenter: "",
      error: null,
    });

    expect(viewModel.center).toBe("semantic drift");
    expect(viewModel.nodeCount).toBe(1);
    expect(viewModel.edgeCount).toBe(1);
    expect(viewModel.tookMsLabel).toBe("37ms");
    expect(viewModel.hasGraph).toBe(true);
    expect(viewModel.nodes[0]?.scoreLabel).toBe("1.000");
    expect(viewModel.edges[0]?.weightLabel).toBe("0.820");
  });

  it("runGraphQuery validates input and delegates to service", async () => {
    const graphMock = vi.mocked(api.fetchKnowledgeGraph);
    const payload: GraphResponse = {
      center: "semantic drift",
      nodes: [],
      edges: [],
      took_ms: 8,
    };
    graphMock.mockResolvedValue(payload);

    await expect(
      runGraphQuery({
        center_concept: "semantic drift",
        visibility: ["shared", "global"],
        depth: 2,
      })
    ).resolves.toEqual(payload);
    expect(graphMock).toHaveBeenCalledWith({
      center_concept: "semantic drift",
      collection: undefined,
      namespaces: undefined,
      visibility: ["shared", "global"],
      tags: undefined,
      depth: 2,
      limit: 50,
      threshold: 0.35,
    });
  });

  it("runGraphQuery rejects missing center concept", async () => {
    await expect(runGraphQuery({})).rejects.toThrow("Center concept is required.");
  });

  it("loadHealth and loadKnowledgeMetrics delegate to services", async () => {
    const healthMock = vi.mocked(api.fetchHealth);
    const metricsMock = vi.mocked(api.fetchKnowledgeHealth);
    const graphMock = vi.mocked(api.fetchKnowledgeGraph);

    const healthPayload: ApiHealthResponse = {
      status: "ok",
      service: "knowledge-observatory",
      timestamp: "2026-01-25T12:00:00Z",
    };
    const metricsPayload: HealthResponse = {
      total_entries: 1,
      collections: [],
      overall_health: "steady",
      overall_metrics: { coherence: 0.5 },
      timestamp: "2026-01-25T10:00:00Z",
    };
    const graphPayload: GraphResponse = {
      center: "semantic drift",
      nodes: [],
      edges: [],
      took_ms: 6,
    };

    healthMock.mockResolvedValue(healthPayload);
    metricsMock.mockResolvedValue(metricsPayload);
    graphMock.mockResolvedValue(graphPayload);

    await expect(loadHealth()).resolves.toEqual(healthPayload);
    await expect(loadKnowledgeMetrics()).resolves.toEqual(metricsPayload);
    await expect(loadKnowledgeGraph({ center_concept: "semantic drift" })).resolves.toEqual(graphPayload);
  });
});
