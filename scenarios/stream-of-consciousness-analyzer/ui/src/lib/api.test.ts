import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiRequestError } from "./api";

// Mock @vrooli/api-base
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

// [REQ:P0-001] API client error handling tests
describe("ApiRequestError", () => {
  it("carries status, category, and retryable fields", () => {
    const err = new ApiRequestError(503, {
      category: "dependency",
      message: "service down",
      retryable: true,
    });
    expect(err.status).toBe(503);
    expect(err.category).toBe("dependency");
    expect(err.retryable).toBe(true);
    expect(err.message).toBe("service down");
    expect(err.name).toBe("ApiRequestError");
  });

  it("extends Error for standard error handling", () => {
    const err = new ApiRequestError(400, {
      category: "validation",
      message: "bad",
      retryable: false,
    });
    expect(err instanceof Error).toBe(true);
    expect(err instanceof ApiRequestError).toBe(true);
  });
});

// [REQ:P0-001] apiFetch error mapping tests
describe("apiFetch error mapping", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("wraps network errors as dependency category", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));

    // Dynamic import to get apiFetch with our mock
    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiRequestError);
      if (!(e instanceof ApiRequestError)) throw e;
      const err = e;
      expect(err.category).toBe("dependency");
      expect(err.retryable).toBe(true);
    }
  });

  it("parses structured API error responses", async () => {
    const apiError = { category: "not_found", message: "scheme not found", retryable: false };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve(apiError),
    });

    const { getScheme } = await import("./api");

    try {
      await getScheme("missing-id");
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      const err = e;
      expect(err.status).toBe(404);
      expect(err.category).toBe("not_found");
      expect(err.message).toBe("scheme not found");
    }
  });

  it("falls back gracefully for non-JSON error responses", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.reject(new Error("not json")),
    });

    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      const err = e;
      expect(err.status).toBe(502);
      expect(err.category).toBe("internal");
      expect(err.retryable).toBe(true);
    }
  });
});

// [REQ:P0-001] Handles non-standard error JSON with "error" field but no "category"
describe("apiFetch error edge cases", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("maps error JSON with 'error' field but missing 'category' to validation", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 422,
      json: () => Promise.resolve({ error: "field is required" }),
    });

    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      expect(e.status).toBe(422);
      expect(e.category).toBe("validation");
      expect(e.message).toBe("field is required");
      expect(e.retryable).toBe(false);
    }
  });

  it("maps 5xx error JSON without category to internal with retryable", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: "something broke" }),
    });

    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      expect(e.category).toBe("internal");
      expect(e.retryable).toBe(true);
      expect(e.message).toBe("something broke");
    }
  });

  it("handles 204 No Content responses without parsing body", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: () => Promise.reject(new Error("no body")),
    });

    const { deleteScheme } = await import("./api");
    // Should not throw even though json() would fail
    await expect(deleteScheme("test-id")).resolves.toBeUndefined();
  });

  it("handles error body that is a non-object JSON value", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: () => Promise.resolve("just a string"),
    });

    const { listSchemes } = await import("./api");

    try {
      await listSchemes();
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      expect(e.category).toBe("internal");
      expect(e.message).toBe("An unexpected error occurred");
    }
  });
});

// [REQ:P0-001] Additional API client function tests
describe("API client functions", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("createEdge sends POST to /thoughts/:id/edges", async () => {
    const mockEdge = { id: "e1", source_id: "t1", target_id: "t2", label: "relates", created_at: "" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockEdge),
    });

    const { createEdge } = await import("./api");
    const result = await createEdge("t1", { target_id: "t2", label: "relates" });

    expect(result.id).toBe("e1");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/thoughts/t1/edges"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("deleteEdge sends DELETE to /thoughts/:id/edges/:edgeId", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: () => Promise.reject(new Error("no body")),
    });

    const { deleteEdge } = await import("./api");
    await expect(deleteEdge("t1", "e1")).resolves.toBeUndefined();
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/thoughts/t1/edges/e1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("exportScheme sends GET to /schemes/:id/export", async () => {
    const mockExport = { scheme: { id: "s1" }, information: [], thoughts: [], edges: [], export_format: "json" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockExport),
    });

    const { exportScheme } = await import("./api");
    const result = await exportScheme("s1");

    expect(result.export_format).toBe("json");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/schemes/s1/export"),
      expect.objectContaining({ headers: { "Content-Type": "application/json" } }),
    );
  });

  it("listProviders sends GET to /providers", async () => {
    const mockProviders = [{ name: "ollama", url: "http://localhost:11434", active: true, fallback: false }];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockProviders),
    });

    const { listProviders } = await import("./api");
    const result = await listProviders();

    expect(result).toHaveLength(1);
    expect(result[0]?.name).toBe("ollama");
  });

  it("fetchHealth returns health status", async () => {
    const mockHealth = { status: "ok", service: "stream-of-consciousness-analyzer", timestamp: "2026-01-01T00:00:00Z" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockHealth),
    });

    const { fetchHealth } = await import("./api");
    const result = await fetchHealth();

    expect(result.status).toBe("ok");
    expect(result.service).toBe("stream-of-consciousness-analyzer");
  });

  it("updateScheme sends PUT with name", async () => {
    const mockScheme = { id: "s1", name: "Renamed", created_at: "", updated_at: "" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockScheme),
    });

    const { updateScheme } = await import("./api");
    const result = await updateScheme("s1", "Renamed");

    expect(result.name).toBe("Renamed");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/schemes/s1"),
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("createInformation sends POST with position data", async () => {
    const mockInfo = { id: "i1", scheme_id: "s1", type: "text", content: "hello", canvas_x: 10, canvas_y: 20, created_at: "", updated_at: "" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockInfo),
    });

    const { createInformation } = await import("./api");
    const result = await createInformation("s1", { type: "text", content: "hello", canvas_x: 10, canvas_y: 20 });

    expect(result.id).toBe("i1");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/schemes/s1/information"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("listThoughts fetches thoughts optionally filtered by scheme", async () => {
    const mockThoughts = [{ id: "t1", scheme_id: "s1", title: "First", body: "", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" }];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockThoughts),
    });

    const { listThoughts } = await import("./api");
    const result = await listThoughts("s1");

    expect(result).toHaveLength(1);
    expect(result[0]?.title).toBe("First");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/thoughts?scheme_id=s1"),
      expect.objectContaining({ headers: { "Content-Type": "application/json" } }),
    );
  });

  it("createThought sends POST to /thoughts", async () => {
    const mockThought = { id: "t-new", scheme_id: "s1", title: "New", body: "body", canvas_x: 10, canvas_y: 20, created_at: "", updated_at: "" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockThought),
    });

    const { createThought } = await import("./api");
    const result = await createThought({ scheme_id: "s1", title: "New", body: "body", canvas_x: 10, canvas_y: 20 });

    expect(result.id).toBe("t-new");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/thoughts"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("updateThought sends PUT to /thoughts/:id", async () => {
    const mockThought = { id: "t1", scheme_id: "s1", title: "Updated", body: "", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockThought),
    });

    const { updateThought } = await import("./api");
    const result = await updateThought("t1", { title: "Updated" });

    expect(result.title).toBe("Updated");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/thoughts/t1"),
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("deleteThought sends DELETE to /thoughts/:id", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: () => Promise.reject(new Error("no body")),
    });

    const { deleteThought } = await import("./api");
    await expect(deleteThought("t1")).resolves.toBeUndefined();
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/thoughts/t1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("listEdges fetches edges for a thought", async () => {
    const mockEdges = [{ id: "e1", source_id: "t1", target_id: "t2", label: "", created_at: "" }];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockEdges),
    });

    const { listEdges } = await import("./api");
    const result = await listEdges("t1");

    expect(result).toHaveLength(1);
    expect(result[0]?.source_id).toBe("t1");
  });
});

// [REQ:P1-001] [REQ:P1-003] Suggestion generation API integration
describe("generateSuggestions", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("calls POST /schemes/:id/suggestions and returns suggestions", async () => {
    const mockResponse = {
      suggestions: [
        { id: "s1", source_id: "t1", target_id: "t2", label: "related", confidence: 0.85, dismissed: false, provider: "ollama" },
      ],
      provider: "ollama",
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockResponse),
    });

    const { generateSuggestions } = await import("./api");
    const result = await generateSuggestions("scheme-1");

    expect(result.suggestions).toHaveLength(1);
    expect(result.suggestions[0]?.label).toBe("related");
    expect(result.provider).toBe("ollama");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/schemes/scheme-1/suggestions"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("throws ApiRequestError when provider is unavailable", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: () => Promise.resolve({ category: "dependency", message: "no LLM provider available", retryable: true }),
    });

    const { generateSuggestions } = await import("./api");

    try {
      await generateSuggestions("scheme-1");
      expect.unreachable("should have thrown");
    } catch (e) {
      if (!(e instanceof ApiRequestError)) throw e;
      expect(e.status).toBe(503);
      expect(e.category).toBe("dependency");
      expect(e.retryable).toBe(true);
    }
  });
});
