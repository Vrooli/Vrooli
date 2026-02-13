import { describe, expect, it, vi } from "vitest";
import { createExecutionPolicyService } from "./execution-policy-service";
import type { IApiClient } from "../lib/api-client";

describe("execution-policy-service", () => {
  it("maps get response", async () => {
    const apiClient = {
      get: vi.fn().mockResolvedValue({
        policy: { default_mode: "manual", default_delay_seconds: 120 },
      }),
      put: vi.fn(),
    } as unknown as IApiClient;
    const service = createExecutionPolicyService(apiClient);

    const policy = await service.get();

    expect(policy.defaultMode).toBe("manual");
    expect(policy.defaultDelaySeconds).toBe(120);
  });

  it("maps update payload", async () => {
    const apiClient = {
      get: vi.fn(),
      put: vi.fn().mockResolvedValue({
        policy: { default_mode: "scheduled", default_delay_seconds: 600 },
      }),
    } as unknown as IApiClient;
    const service = createExecutionPolicyService(apiClient);

    const policy = await service.update({ defaultMode: "scheduled", defaultDelaySeconds: 600 });

    expect(apiClient.put).toHaveBeenCalled();
    expect(policy.defaultMode).toBe("scheduled");
    expect(policy.defaultDelaySeconds).toBe(600);
  });
});
