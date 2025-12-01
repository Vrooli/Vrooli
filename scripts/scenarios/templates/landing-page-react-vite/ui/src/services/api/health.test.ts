import { describe, it, expect, beforeEach, vi } from "vitest";
import { fetchHealth } from "./health";

describe("API utilities", () => {
  beforeEach(() => {
    // Clear all mocks before each test
    vi.clearAllMocks();
  });

  describe("fetchHealth", () => {
    it("should successfully fetch health status", async () => {
      // Mock successful response
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          status: "healthy",
          service: "landing-manager",
          timestamp: "2025-11-21T00:00:00Z",
        }),
      });

      const result = await fetchHealth();

      expect(result).toEqual({
        status: "healthy",
        service: "landing-manager",
        timestamp: "2025-11-21T00:00:00Z",
      });
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    it("should throw error when API returns non-ok status", async () => {
      // Mock failed response
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: async () => "Internal Server Error",
      });

      await expect(fetchHealth()).rejects.toThrow("API call failed (500): Internal Server Error");
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    it("should include correct headers", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ status: "healthy", service: "landing-manager", timestamp: "" }),
      });

      await fetchHealth();

      const callArgs = (global.fetch as any).mock.calls[0];
      expect(callArgs[1].headers["Content-Type"]).toBe("application/json");
      expect(callArgs[1].credentials).toBe("include");
    });
  });
});
