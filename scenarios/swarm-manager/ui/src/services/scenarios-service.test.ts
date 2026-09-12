import { describe, it, expect, vi, beforeEach } from "vitest";
import { createScenariosService, type IScenariosService } from "./scenarios-service";
import type { IApiClient } from "../lib/api-client";
import type { Scenario, UpdateScenarioMetadataRequest } from "../types";

describe("Scenarios Service", () => {
  let mockApiClient: IApiClient;
  let service: IScenariosService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createScenariosService(mockApiClient);
  });

  it("lists scenarios", async () => {
    const mockScenarios: Scenario[] = [
      {
        name: "test-scenario",
        displayName: "Test Scenario",
        description: "A test scenario",
        status: "running",
        priority: 1,
        isGreenfield: false,
        tags: ["test"],
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
        },
      ],
    });

    const result = await service.list();
    expect(mockApiClient.get).toHaveBeenCalledWith("/scenarios");
    expect(result).toEqual(mockScenarios);
  });

  it("gets a single scenario", async () => {
    const mockScenario: Scenario = {
      name: "my-scenario",
      displayName: "My Scenario",
      description: "A specific scenario",
      status: "stopped",
      priority: 2,
      isGreenfield: true,
      tags: ["important"],
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
      },
    });

    const result = await service.get("my-scenario");
    expect(mockApiClient.get).toHaveBeenCalledWith("/scenarios/my-scenario");
    expect(result).toEqual(mockScenario);
  });

  it("normalizes scenario coverage from the goal-based API contract", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      scenario_name: "api-server",
      goals: [{
        name: "reliability",
        title: "API Reliability",
        status: "active",
        priority: 4,
        scope: { total: 2, completed: 1, in_progress: 1, failed: 0, pending: 0, archived: 0 },
      }],
      orphan_items: [],
      rollup: { total: 2, completed: 1, in_progress: 1, failed: 0, pending: 0, archived: 0 },
      fixes: { active: [{ name: "timeout", title: "Timeout", goal: "reliability", path: "fix/timeout" }], archived: [] },
    });

    await expect(service.getContext("api-server")).resolves.toMatchObject({
      scenarioName: "api-server",
      goals: [{ name: "reliability", title: "API Reliability", rollup: { total: 2, inProgress: 1 } }],
      fixes: { active: [{ goal: "reliability" }] },
    });
    expect(mockApiClient.get).toHaveBeenCalledWith("/scenarios/api-server/context");
  });

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
      },
    });

    const result = await service.updateMetadata("my-scenario", request);

    expect(mockApiClient.patch).toHaveBeenCalledWith("/scenarios/my-scenario", {
      is_greenfield: true,
    });
    expect(result).toEqual(updatedScenario);
  });

  it("supports lifecycle actions", async () => {
    const mockScenario: Scenario = {
      name: "api-server",
      displayName: "API Server",
      description: "Backend service",
      status: "running",
      priority: 1,
      isGreenfield: false,
      tags: [],
    };

    vi.mocked(mockApiClient.post).mockResolvedValue({
      scenario: {
        name: "api-server",
        display_name: "API Server",
        description: "Backend service",
        status: "running",
        priority: 1,
        is_greenfield: false,
        tags: [],
      },
    });

    await expect(service.start("api-server")).resolves.toEqual(mockScenario);
    expect(mockApiClient.post).toHaveBeenCalledWith("/scenarios/api-server/start", {});

    await expect(service.stop("api-server")).resolves.toEqual(mockScenario);
    expect(mockApiClient.post).toHaveBeenCalledWith("/scenarios/api-server/stop", {});

    await expect(service.restart("api-server")).resolves.toEqual(mockScenario);
    expect(mockApiClient.post).toHaveBeenCalledWith("/scenarios/api-server/restart", {});
  });
});
