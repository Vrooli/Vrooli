import { describe, it, expect, beforeEach, vi } from "vitest";
import { fetchHealth } from "./api";

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
      });

      await expect(fetchHealth()).rejects.toThrow("API health check failed: 500");
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    it("should include correct headers", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ status: "healthy", service: "landing-manager", timestamp: "" }),
      });

      await fetchHealth();

      const callArgs = (global.fetch as any).mock.calls[0];
      expect(callArgs[1].headers).toEqual({ "Content-Type": "application/json" });
      expect(callArgs[1].cache).toBe("no-store");
    });
  });
});
