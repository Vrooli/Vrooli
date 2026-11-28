import { describe, it, expect, vi, beforeEach } from "vitest";
import { createAutoDrainService, type IAutoDrainService } from "./auto-drain-service";
import type { IApiClient } from "../lib/api-client";

describe("Auto-drain service", () => {
  let mockApiClient: IApiClient;
  let service: IAutoDrainService;

  beforeEach(() => {
    mockApiClient = { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() };
    service = createAutoDrainService(mockApiClient);
  });

  it("reads the toggle, coercing a missing flag to false", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({});
    expect(await service.get()).toEqual({ enabled: false });
    expect(mockApiClient.get).toHaveBeenCalledWith("/execution/auto-drain");
  });

  it("writes the toggle via PUT and normalizes the echo", async () => {
    vi.mocked(mockApiClient.put).mockResolvedValue({ enabled: true });
    expect(await service.set(true)).toEqual({ enabled: true });
    expect(mockApiClient.put).toHaveBeenCalledWith("/execution/auto-drain", { enabled: true });
  });
});
