import { describe, it, expect, vi, beforeEach } from "vitest";
import { createScenariosService, type IScenariosService } from "./scenarios-service";
import type { IApiClient } from "../lib/api-client";
import type { Scenario, UpdateScenarioMetadataRequest } from "../types";

/**
 * Scenarios Service Tests
 *
 * These tests demonstrate the seam pattern - by injecting a mock API client,
 * we can test the service layer in isolation without needing to mock HTTP.
 *
 * This is cleaner than module-level mocking because:
 * 1. Tests explicitly show their dependencies
 * 2. No magic mocking of import paths
 * 3. Easy to test with different client configurations
 *
 * DOC: docs/internal/SEAMS.md#ui-to-api-seam-improved-in-phase-3
 */

// [REQ:MOD-P0-006] Test scenarios service seam
describe("Scenarios Service", () => {
  let mockApiClient: IApiClient;
  let service: IScenariosService;

  beforeEach(() => {
    // Create a mock API client - the seam allows easy substitution
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };

    // Inject the mock client into the service
    service = createScenariosService(mockApiClient);
  });

  describe("list", () => {
    it("fetches scenarios from the correct endpoint", async () => {
      const mockScenarios: Scenario[] = [
        {
          name: "test-scenario",
          displayName: "Test Scenario",
          description: "A test scenario",
          status: "running",
          priority: 1,
          isGreenfield: false,
          tags: ["test"],
          recommendationsEnabled: true,
        },
      ];
      vi.mocked(mockApiClient.get).mockResolvedValue({
        scenarios: [
          {
            name: "test-scenario",
            display_name: "Test Scenario",
            description: "A test scenario",
            status: "running",
            priority: 1,
            is_greenfield: false,
            tags: ["test"],
            recommendations_enabled: true,
          },
        ],
      });

      const result = await service.list();

      expect(mockApiClient.get).toHaveBeenCalledWith("/scenarios");
      expect(result).toEqual(mockScenarios);
    });

    it("returns empty array when no scenarios exist", async () => {
      vi.mocked(mockApiClient.get).mockResolvedValue({ scenarios: [] });

      const result = await service.list();

      expect(mockApiClient.get).toHaveBeenCalledWith("/scenarios");
      expect(result).toEqual([]);
    });
  });

  describe("get", () => {
    it("fetches a single scenario by name", async () => {
      const mockScenario: Scenario = {
        name: "my-scenario",
        displayName: "My Scenario",
        description: "A specific scenario",
        status: "stopped",
        priority: 2,
        isGreenfield: true,
        tags: ["important"],
        recommendationsEnabled: false,
      };
      vi.mocked(mockApiClient.get).mockResolvedValue({
        scenario: {
          name: "my-scenario",
          display_name: "My Scenario",
          description: "A specific scenario",
          status: "stopped",
          priority: 2,
          is_greenfield: true,
          tags: ["important"],
          recommendations_enabled: false,
        },
      });

      const result = await service.get("my-scenario");

      expect(mockApiClient.get).toHaveBeenCalledWith("/scenarios/my-scenario");
      expect(result).toEqual(mockScenario);
    });
  });

  describe("updateMetadata", () => {
    it("patches scenario metadata with isGreenfield", async () => {
      const request: UpdateScenarioMetadataRequest = {
        isGreenfield: true,
      };
      const updatedScenario: Scenario = {
        name: "my-scenario",
        displayName: "My Scenario",
        description: "Updated scenario",
        status: "running",
        priority: 1,
        isGreenfield: true,
        tags: [],
        recommendationsEnabled: true,
      };
      vi.mocked(mockApiClient.patch).mockResolvedValue({
        scenario: {
          name: "my-scenario",
          display_name: "My Scenario",
          description: "Updated scenario",
          status: "running",
          priority: 1,
          is_greenfield: true,
          tags: [],
          recommendations_enabled: true,
        },
      });

      const result = await service.updateMetadata("my-scenario", request);

      expect(mockApiClient.patch).toHaveBeenCalledWith("/scenarios/my-scenario", {
        is_greenfield: true,
      });
      expect(result).toEqual(updatedScenario);
      expect(result.isGreenfield).toBe(true);
    });

    it("patches scenario metadata with recommendationsEnabled", async () => {
      const request: UpdateScenarioMetadataRequest = {
        recommendationsEnabled: false,
      };
      const updatedScenario: Scenario = {
        name: "my-scenario",
        displayName: "My Scenario",
        description: "Updated scenario",
        status: "running",
        priority: 1,
        isGreenfield: false,
        tags: [],
        recommendationsEnabled: false,
      };
      vi.mocked(mockApiClient.patch).mockResolvedValue({
        scenario: {
          name: "my-scenario",
          display_name: "My Scenario",
          description: "Updated scenario",
          status: "running",
          priority: 1,
          is_greenfield: false,
          tags: [],
          recommendations_enabled: false,
        },
      });

      const result = await service.updateMetadata("my-scenario", request);

      expect(mockApiClient.patch).toHaveBeenCalledWith("/scenarios/my-scenario", {
        recommendations_enabled: false,
      });
      expect(result).toEqual(updatedScenario);
      expect(result.recommendationsEnabled).toBe(false);
    });

    it("patches scenario metadata with both fields", async () => {
      const request: UpdateScenarioMetadataRequest = {
        isGreenfield: true,
        recommendationsEnabled: false,
      };
      const updatedScenario: Scenario = {
        name: "my-scenario",
        displayName: "My Scenario",
        description: "Updated scenario",
        status: "stopped",
        priority: 3,
        isGreenfield: true,
        tags: ["updated"],
        recommendationsEnabled: false,
      };
      vi.mocked(mockApiClient.patch).mockResolvedValue({
        scenario: {
          name: "my-scenario",
          display_name: "My Scenario",
          description: "Updated scenario",
          status: "stopped",
          priority: 3,
          is_greenfield: true,
          tags: ["updated"],
          recommendations_enabled: false,
        },
      });

      const result = await service.updateMetadata("my-scenario", request);

      expect(mockApiClient.patch).toHaveBeenCalledWith("/scenarios/my-scenario", {
        is_greenfield: true,
        recommendations_enabled: false,
      });
      expect(result).toEqual(updatedScenario);
    });
  });

  describe("error handling", () => {
    it("propagates API errors from list", async () => {
      const error = new Error("Network error");
      vi.mocked(mockApiClient.get).mockRejectedValue(error);

      await expect(service.list()).rejects.toThrow("Network error");
    });

    it("propagates API errors from get", async () => {
      const error = new Error("Not found");
      vi.mocked(mockApiClient.get).mockRejectedValue(error);

      await expect(service.get("missing-scenario")).rejects.toThrow("Not found");
    });

    it("propagates API errors from updateMetadata", async () => {
      const error = new Error("Bad request");
      vi.mocked(mockApiClient.patch).mockRejectedValue(error);

      await expect(
        service.updateMetadata("my-scenario", { isGreenfield: true })
      ).rejects.toThrow("Bad request");
    });
  });
});
