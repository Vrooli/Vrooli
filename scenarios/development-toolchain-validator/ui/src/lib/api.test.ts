// [REQ:REQ-P0-002] Reference Scenario API Endpoints - Client tests
// [REQ:REQ-P0-003] Skill Connection Management - Client tests
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchHealth,
  fetchReferences,
  fetchReferenceById,
  fetchReferenceBySlug,
  createReference,
  updateReference,
  deleteReference,
  fetchConnections,
  fetchConnectionsByReference,
  type Reference,
  type SkillConnection
} from "./api";

// Mock @vrooli/api-base module
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:16980/api/v1",
  buildApiUrl: (path: string, _opts: { baseUrl: string }) => `http://localhost:16980/api/v1${path}`
}));

// Helper to create mock fetch response
function mockFetchResponse(data: unknown, ok = true, status = 200) {
  return Promise.resolve({
    ok,
    status,
    json: () => Promise.resolve(data)
  } as Response);
}

// Helper to create mock reference
function createMockReference(overrides: Partial<Reference> = {}): Reference {
  return {
    id: "test-id-123",
    slug: "test-ref",
    name: "Test Reference",
    template: "react-vite",
    path: "/scenarios/test-ref",
    description: "A test reference",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides
  };
}

// Helper to create mock connection
function createMockConnection(overrides: Partial<SkillConnection> = {}): SkillConnection {
  return {
    id: "conn-id-123",
    reference_id: "ref-id-123",
    skill_id: "skill-id-123",
    skill_version: "1.0.0",
    skill_content_hash: "abc123",
    connected_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides
  };
}

describe("API Client", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Health endpoint tests
  // ─────────────────────────────────────────────────────────────────────────
  describe("fetchHealth", () => {
    it("returns health response on success", async () => {
      // ARRANGE
      const mockHealth = {
        status: "healthy",
        service: "development-toolchain-validator",
        timestamp: "2024-01-01T00:00:00Z"
      };
      vi.mocked(fetch).mockImplementation(() => mockFetchResponse(mockHealth));

      // ACT
      const result = await fetchHealth();

      // ASSERT
      expect(result).toEqual(mockHealth);
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/health",
        expect.objectContaining({
          method: "GET",
          headers: { "Content-Type": "application/json" }
        })
      );
    });

    it("throws error on API failure", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ error: "Service unavailable" }, false, 503)
      );

      // ACT & ASSERT
      await expect(fetchHealth()).rejects.toThrow("Service unavailable");
    });
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Reference CRUD tests
  // ─────────────────────────────────────────────────────────────────────────
  describe("fetchReferences", () => {
    it("returns list of references", async () => {
      // ARRANGE
      const mockRefs = [createMockReference(), createMockReference({ id: "ref-2" })];
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ references: mockRefs, count: 2 })
      );

      // ACT
      const result = await fetchReferences();

      // ASSERT
      expect(result).toHaveLength(2);
      expect(result[0]?.slug).toBe("test-ref");
    });

    it("includes template filter in query string", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ references: [], count: 0 })
      );

      // ACT
      await fetchReferences("react-vite");

      // ASSERT
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("template=react-vite"),
        expect.any(Object)
      );
    });

    it("returns empty array when no references exist", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ references: [], count: 0 })
      );

      // ACT
      const result = await fetchReferences();

      // ASSERT
      expect(result).toEqual([]);
    });
  });

  describe("fetchReferenceById", () => {
    it("returns reference by ID", async () => {
      // ARRANGE
      const mockRef = createMockReference();
      vi.mocked(fetch).mockImplementation(() => mockFetchResponse(mockRef));

      // ACT
      const result = await fetchReferenceById("test-id-123");

      // ASSERT
      expect(result.id).toBe("test-id-123");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/references/test-id-123",
        expect.any(Object)
      );
    });

    it("throws error when reference not found", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ error: "Reference not found" }, false, 404)
      );

      // ACT & ASSERT
      await expect(fetchReferenceById("nonexistent")).rejects.toThrow("Reference not found");
    });
  });

  describe("fetchReferenceBySlug", () => {
    it("returns reference by slug", async () => {
      // ARRANGE
      const mockRef = createMockReference({ slug: "my-ref" });
      vi.mocked(fetch).mockImplementation(() => mockFetchResponse(mockRef));

      // ACT
      const result = await fetchReferenceBySlug("my-ref");

      // ASSERT
      expect(result.slug).toBe("my-ref");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/references/by-slug/my-ref",
        expect.any(Object)
      );
    });

    it("encodes special characters in slug", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse(createMockReference())
      );

      // ACT
      await fetchReferenceBySlug("slug with spaces");

      // ASSERT
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("slug%20with%20spaces"),
        expect.any(Object)
      );
    });
  });

  describe("createReference", () => {
    it("creates reference with valid input", async () => {
      // ARRANGE
      const input = {
        slug: "new-ref",
        name: "New Reference",
        template: "react-vite",
        path: "/scenarios/new-ref"
      };
      const mockRef = createMockReference(input);
      vi.mocked(fetch).mockImplementation(() => mockFetchResponse(mockRef));

      // ACT
      const result = await createReference(input);

      // ASSERT
      expect(result.slug).toBe("new-ref");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/references",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify(input)
        })
      );
    });

    it("throws error on validation failure", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ error: "Invalid slug format" }, false, 400)
      );

      // ACT & ASSERT
      await expect(
        createReference({
          slug: "invalid slug!",
          name: "Test",
          template: "react-vite",
          path: "/test"
        })
      ).rejects.toThrow("Invalid slug format");
    });
  });

  describe("updateReference", () => {
    it("updates reference with partial fields", async () => {
      // ARRANGE
      const input = { name: "Updated Name" };
      const mockRef = createMockReference({ name: "Updated Name" });
      vi.mocked(fetch).mockImplementation(() => mockFetchResponse(mockRef));

      // ACT
      const result = await updateReference("test-id", input);

      // ASSERT
      expect(result.name).toBe("Updated Name");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/references/test-id",
        expect.objectContaining({
          method: "PATCH",
          body: JSON.stringify(input)
        })
      );
    });
  });

  describe("deleteReference", () => {
    it("deletes reference successfully", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        Promise.resolve({
          ok: true,
          status: 204,
          json: () => Promise.reject(new Error("No content"))
        } as Response)
      );

      // ACT & ASSERT
      await expect(deleteReference("test-id")).resolves.toBeUndefined();
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/references/test-id",
        expect.objectContaining({ method: "DELETE" })
      );
    });

    it("throws error when reference not found", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ error: "Reference not found" }, false, 404)
      );

      // ACT & ASSERT
      await expect(deleteReference("nonexistent")).rejects.toThrow("Reference not found");
    });
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Skill Connection tests
  // ─────────────────────────────────────────────────────────────────────────
  describe("fetchConnections", () => {
    it("returns all connections without filter", async () => {
      // ARRANGE
      const mockConns = [createMockConnection(), createMockConnection({ id: "conn-2" })];
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ connections: mockConns, count: 2 })
      );

      // ACT
      const result = await fetchConnections();

      // ASSERT
      expect(result).toHaveLength(2);
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:16980/api/v1/connections",
        expect.any(Object)
      );
    });

    it("includes reference_id filter in query string", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ connections: [], count: 0 })
      );

      // ACT
      await fetchConnections("ref-123");

      // ASSERT
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("reference_id=ref-123"),
        expect.any(Object)
      );
    });
  });

  describe("fetchConnectionsByReference", () => {
    it("is an alias for fetchConnections with referenceId", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({ connections: [], count: 0 })
      );

      // ACT
      await fetchConnectionsByReference("ref-123");

      // ASSERT
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("reference_id=ref-123"),
        expect.any(Object)
      );
    });
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Error handling edge cases
  // ─────────────────────────────────────────────────────────────────────────
  describe("error handling", () => {
    it("handles malformed JSON error response gracefully", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        Promise.resolve({
          ok: false,
          status: 500,
          json: () => Promise.reject(new SyntaxError("Unexpected token"))
        } as Response)
      );

      // ACT & ASSERT
      await expect(fetchHealth()).rejects.toThrow("Unknown error");
    });

    it("uses fallback when error object has no error field", async () => {
      // ARRANGE
      vi.mocked(fetch).mockImplementation(() =>
        mockFetchResponse({}, false, 500)
      );

      // ACT & ASSERT
      await expect(fetchHealth()).rejects.toThrow("API health check failed: 500");
    });
  });
});
