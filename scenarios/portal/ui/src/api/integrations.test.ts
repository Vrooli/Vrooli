import { describe, expect, it, vi } from "vitest";
import { BehaviorMode, IntegrationState } from "@vrooli/proto-types/portal/v1/shared/common_pb";
import { BehaviorOverride } from "@vrooli/proto-types/portal/v1/integrations/integrations_pb";

const client = vi.hoisted(() => ({
  status: vi.fn(),
  updateOverride: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => client),
}));

import {
  behaviorModeLabel,
  fetchIntegrationsStatus,
  integrationStateLabel,
  updateBehaviorOverride,
} from "./integrations";

describe("api/integrations Connect wrappers", () => {
  it("fetches status and falls back after override responses without status", async () => {
    client.status.mockResolvedValueOnce({ activeMode: BehaviorMode.PASSIVE });
    client.status.mockResolvedValueOnce({ activeMode: BehaviorMode.OFF });
    client.updateOverride.mockResolvedValueOnce({});

    await expect(fetchIntegrationsStatus()).resolves.toEqual({ activeMode: BehaviorMode.PASSIVE });
    await expect(updateBehaviorOverride(BehaviorOverride.FORCE_OFF)).resolves.toEqual({
      activeMode: BehaviorMode.OFF,
    });

    expect(client.status).toHaveBeenCalledWith({});
    expect(client.updateOverride).toHaveBeenCalledWith({ override: BehaviorOverride.FORCE_OFF });
  });

  it("labels known and unknown behavior modes and integration states", () => {
    expect(behaviorModeLabel(BehaviorMode.OFF)).toBe("off");
    expect(behaviorModeLabel(BehaviorMode.PASSIVE)).toBe("passive");
    expect(behaviorModeLabel(BehaviorMode.FULL)).toBe("full");
    expect(behaviorModeLabel(BehaviorMode.UNSPECIFIED)).toBe("unknown");
    expect(integrationStateLabel(IntegrationState.AVAILABLE)).toBe("available");
    expect(integrationStateLabel(IntegrationState.DEGRADED)).toBe("degraded");
    expect(integrationStateLabel(IntegrationState.UNAVAILABLE)).toBe("unavailable");
    expect(integrationStateLabel(IntegrationState.UNKNOWN)).toBe("unknown");
    expect(integrationStateLabel(IntegrationState.UNSPECIFIED)).toBe("unspecified");
  });
});
