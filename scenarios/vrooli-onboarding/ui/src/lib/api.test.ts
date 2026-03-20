import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// Mock @vrooli/api-base before importing api module
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) => `${opts.baseUrl}${path}`,
}));

import {
  fetchHealth,
  fetchResources,
  fetchProgress,
  updateProgress,
  generateConfig,
  validateConfig,
  fetchResourceHealth,
  fetchGlossary,
  fetchSetupOrder,
} from "./api";

function mockFetch(body: unknown, ok = true, status = 200) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok,
    status,
    json: () => Promise.resolve(body),
  });
}

describe("api", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  // [REQ:P0-001] Resource Discovery API
  describe("fetchResources", () => {
    it("unwraps wrapped response with resources array", async () => {
      const resources = [{ name: "postgres", status: "enabled", category: "storage", installed: "yes", last_updated: "2026-01-01" }];
      mockFetch({ count: 1, resources });

      const result = await fetchResources();
      expect(result).toEqual(resources);
    });

    it("returns array directly when API returns plain array", async () => {
      const resources = [{ name: "redis", status: "enabled", category: "storage", installed: "yes", last_updated: "2026-01-01" }];
      mockFetch(resources);

      const result = await fetchResources();
      expect(result).toEqual(resources);
    });

    it("throws on non-ok response", async () => {
      mockFetch(null, false, 500);
      await expect(fetchResources()).rejects.toThrow("API request failed: 500");
    });

    it("sends correct URL and headers", async () => {
      mockFetch({ resources: [] });
      await fetchResources();

      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/resources",
        expect.objectContaining({
          headers: { "Content-Type": "application/json" },
          cache: "no-store",
        }),
      );
    });
  });

  // [REQ:P1-001] Resource Health API
  describe("fetchResourceHealth", () => {
    it("returns resource health response", async () => {
      const healthData = { resources: [], total: 0, healthy_count: 0, checked_at: "2026-01-01T00:00:00Z" };
      mockFetch(healthData);

      const result = await fetchResourceHealth();
      expect(result).toEqual(healthData);
    });

    it("throws on server error", async () => {
      mockFetch(null, false, 503);
      await expect(fetchResourceHealth()).rejects.toThrow("API request failed: 503");
    });
  });

  // [REQ:P0-004] Config Generation
  describe("generateConfig", () => {
    it("sends POST with resource names", async () => {
      const config = { postgres: { host: "localhost" } };
      mockFetch(config);

      const result = await generateConfig(["postgres", "redis"]);
      expect(result).toEqual(config);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/config/generate",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ resources: ["postgres", "redis"] }),
        }),
      );
    });

    it("handles empty resource list", async () => {
      mockFetch({});
      await generateConfig([]);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ body: JSON.stringify({ resources: [] }) }),
      );
    });
  });

  // [REQ:P0-005] Config Validation
  describe("validateConfig", () => {
    it("sends POST with resource config", async () => {
      const validationResult = { valid: true };
      mockFetch(validationResult);

      const resources = { postgres: { enabled: true, name: "PostgreSQL" } };
      const result = await validateConfig(resources);
      expect(result).toEqual(validationResult);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/config/validate",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ resources }),
        }),
      );
    });

    it("returns validation errors", async () => {
      const validationResult = { valid: false, errors: ["Missing required field"], warnings: ["Deprecated option"] };
      mockFetch(validationResult);

      const result = await validateConfig({});
      expect(result.valid).toBe(false);
      expect(result.errors).toContain("Missing required field");
      expect(result.warnings).toContain("Deprecated option");
    });
  });

  // [REQ:P1-003] Progress Storage
  describe("fetchProgress", () => {
    it("fetches progress without user_id", async () => {
      const progress = { id: 1, user_id: "default", current_step: 2, completed_steps: [0, 1], config_data: {}, updated_at: "2026-01-01" };
      mockFetch(progress);

      const result = await fetchProgress();
      expect(result).toEqual(progress);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/progress",
        expect.any(Object),
      );
    });

    it("includes user_id query param when provided", async () => {
      mockFetch({ id: 1, user_id: "user-123", current_step: 0, completed_steps: [], config_data: {}, updated_at: "" });

      await fetchProgress("user-123");
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/progress?user_id=user-123",
        expect.any(Object),
      );
    });
  });

  describe("updateProgress", () => {
    it("sends PUT with progress data and default user_id", async () => {
      const updated = { id: 1, user_id: "default", current_step: 3, completed_steps: [0, 1, 2], config_data: { key: "val" }, updated_at: "2026-01-01" };
      mockFetch(updated);

      const result = await updateProgress({ current_step: 3, completed_steps: [0, 1, 2], config_data: { key: "val" } });
      expect(result).toEqual(updated);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/progress",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({ user_id: "default", current_step: 3, completed_steps: [0, 1, 2], config_data: { key: "val" } }),
        }),
      );
    });
  });

  // [REQ:P2-002] Technical Glossary
  describe("fetchGlossary", () => {
    it("fetches all glossary entries without query", async () => {
      const glossary = { entries: [{ term: "Docker", description: "Container runtime", category: "infrastructure" }], count: 1 };
      mockFetch(glossary);

      const result = await fetchGlossary();
      expect(result).toEqual(glossary);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/glossary",
        expect.any(Object),
      );
    });

    it("includes search query param when provided", async () => {
      mockFetch({ entries: [], count: 0, query: "docker" });

      await fetchGlossary("docker");
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "http://localhost:9999/api/v1/glossary?q=docker",
        expect.any(Object),
      );
    });

    it("encodes special characters in query", async () => {
      mockFetch({ entries: [], count: 0 });

      await fetchGlossary("hello world&foo=bar");
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("q=hello%20world%26foo%3Dbar"),
        expect.any(Object),
      );
    });
  });

  // [REQ:P2-001] Setup Order Algorithm
  describe("fetchSetupOrder", () => {
    it("returns setup order response", async () => {
      const order = { setup_order: [{ name: "postgres", category: "storage", order: 1, dependencies: [] }], total: 1 };
      mockFetch(order);

      const result = await fetchSetupOrder();
      expect(result).toEqual(order);
    });
  });

  describe("fetchHealth", () => {
    it("returns health status", async () => {
      const health = { status: "healthy", service: "vrooli-onboarding", timestamp: "2026-01-01T00:00:00Z" };
      mockFetch(health);

      const result = await fetchHealth();
      expect(result).toEqual(health);
    });

    it("throws on 404", async () => {
      mockFetch(null, false, 404);
      await expect(fetchHealth()).rejects.toThrow("API request failed: 404");
    });
  });

  describe("typedFetch error handling", () => {
    it("throws with status code in message for various HTTP errors", async () => {
      for (const status of [400, 401, 403, 404, 500, 502, 503]) {
        mockFetch(null, false, status);
        await expect(fetchHealth()).rejects.toThrow(`API request failed: ${status}`);
      }
    });

    it("sets Content-Type header on all requests", async () => {
      mockFetch({ status: "ok" });
      await fetchHealth();
      const callArgs = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(callArgs[1].headers).toEqual({ "Content-Type": "application/json" });
    });
  });
});
