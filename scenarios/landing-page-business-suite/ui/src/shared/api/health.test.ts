import { describe, it, expect, beforeEach, vi } from "vitest";
import { fetchHealth } from "./health";
import { createFetchMock, mockResponses, installFetchMock, getFetchCall } from "../test-utils/api-mocks";

describe("API utilities", () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    // Clear all mocks before each test
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  describe("fetchHealth", () => {
    it("should successfully fetch health status", async () => {
      // Mock successful response
      fetchMock.mockResolvedValue(mockResponses.success({
        status: "healthy",
        service: "landing-manager",
        timestamp: "2025-11-21T00:00:00Z",
      }));

      const result = await fetchHealth();

      expect(result).toEqual({
        status: "healthy",
        service: "landing-manager",
        timestamp: "2025-11-21T00:00:00Z",
      });
      expect(globalThis.fetch).toHaveBeenCalledTimes(1);
    });

    it("should throw error when API returns non-ok status", async () => {
      // Mock failed response
      fetchMock.mockResolvedValue(mockResponses.error(500, "Internal Server Error"));

      await expect(fetchHealth()).rejects.toThrow('API call failed (500): {"error":"Internal Server Error"}');
      expect(globalThis.fetch).toHaveBeenCalledTimes(1);
    });

    it("should include correct headers", async () => {
      fetchMock.mockResolvedValue(mockResponses.success({
        status: "healthy",
        service: "landing-manager",
        timestamp: "",
      }));

      await fetchHealth();

      const [, options] = getFetchCall(fetchMock);
      const headers = new Headers(options.headers);
      expect(headers.get("Content-Type")).toBe("application/json");
      expect(options.credentials).toBe("include");
    });
  });
});
