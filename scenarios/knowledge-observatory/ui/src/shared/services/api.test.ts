import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { fetchHealth, fetchKnowledgeGraph, fetchKnowledgeHealth, searchKnowledge } from "./api";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://test-base",
  buildApiUrl: (path: string, { baseUrl }: { baseUrl: string }) => `${baseUrl}${path}`,
}));

const createResponse = (body: unknown, options?: { ok?: boolean; status?: number }) =>
  ({
    ok: options?.ok ?? true,
    status: options?.status ?? 200,
    json: vi.fn().mockResolvedValue(body),
  }) as unknown as Response;

describe("services/api", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fetchHealth normalizes missing fields", async () => {
    fetchMock.mockResolvedValue(
      createResponse({ status: "ok", service: "svc", timestamp: "2026-01-25T10:00:00Z" })
    );

    const result = await fetchHealth();

    expect(result.status).toBe("ok");
    expect(result.service).toBe("svc");
    expect(typeof result.timestamp).toBe("string");
  });

  it("fetchHealth throws on non-OK responses", async () => {
    fetchMock.mockResolvedValue(createResponse({ error: "boom" }, { ok: false, status: 500 }));

    await expect(fetchHealth()).rejects.toThrow("API health check failed: 500");
  });

  it("searchKnowledge normalizes results and defaults", async () => {
    fetchMock.mockResolvedValue(
      createResponse({
        results: [
          { id: "", score: 0.4, content: "", metadata: {} },
          { id: "r-2", score: 0.4, content: "content", metadata: { source: "alpha" } },
        ],
        query: "",
        took_ms: 12,
      })
    );

    const result = await searchKnowledge({ query: "semantic query", limit: 10 });

    expect(result.results).toHaveLength(2);
    expect(result.results[0]?.id).toBe("result-1");
    expect(result.results[0]?.score).toBe(0.4);
    expect(result.results[0]?.metadata).toEqual({});
    expect(result.query).toBe("semantic query");
    expect(result.took_ms).toBe(12);
  });

  it("searchKnowledge throws on invalid response shapes", async () => {
    fetchMock.mockResolvedValue(
      createResponse({
        results: "nope",
        query: 42,
        took_ms: "invalid",
      })
    );

    await expect(searchKnowledge({ query: "semantic query" })).rejects.toThrow("Invalid search response");
  });

  it("searchKnowledge surfaces API error messages", async () => {
    fetchMock.mockResolvedValue(createResponse({ error: "Search failed" }, { ok: false, status: 400 }));

    await expect(searchKnowledge({ query: "q" })).rejects.toThrow("Search failed");
  });

  it("fetchKnowledgeHealth normalizes collections and defaults", async () => {
    fetchMock.mockResolvedValue(
      createResponse({
        collections: [{ name: "alpha", size: 11, metrics: { coherence: 0.7 } }],
        overall_health: "steady",
        overall_metrics: { freshness: 0.8 },
        timestamp: "2026-01-25T10:00:00Z",
      })
    );

    const result = await fetchKnowledgeHealth();

    expect(result.collections).toHaveLength(1);
    expect(result.collections[0]?.name).toBe("alpha");
    expect(result.collections[0]?.size).toBe(11);
    expect(result.overall_health).toBe("steady");
    expect(typeof result.timestamp).toBe("string");
  });

  it("fetchKnowledgeGraph normalizes nodes, edges, and defaults", async () => {
    fetchMock.mockResolvedValue(
      createResponse({
        center: "",
        nodes: [{ id: "", label: "", score: 0.91, metadata: { source: "docs" } }],
        edges: [{ source: "center", target: "node-a", weight: 0.7, relationship: "" }],
        took_ms: 19,
      })
    );

    const result = await fetchKnowledgeGraph({ center_concept: "semantic drift", limit: 10 });

    expect(result.center).toBe("semantic drift");
    expect(result.nodes).toHaveLength(1);
    expect(result.nodes[0]?.id).toBe("node-1");
    expect(result.nodes[0]?.label).toBe("node-1");
    expect(result.nodes[0]?.metadata).toEqual({ source: "docs" });
    expect(result.edges).toHaveLength(1);
    expect(result.edges[0]?.relationship).toBe("semantic_similarity");
    expect(result.took_ms).toBe(19);
  });

  it("fetchKnowledgeGraph surfaces API error messages", async () => {
    fetchMock.mockResolvedValue(createResponse({ error: "Graph failed" }, { ok: false, status: 400 }));

    await expect(fetchKnowledgeGraph({ center_concept: "alpha" })).rejects.toThrow("Graph failed");
  });
});
