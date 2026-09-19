import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BehaviorOverride } from "@vrooli/proto-types/portal/v1/integrations/integrations_pb";
import { BehaviorMode } from "@vrooli/proto-types/portal/v1/shared/common_pb";

import { i18n } from "../../i18n";
import { useIntegrationStatus } from "./useIntegrationStatus";

const integrationsApi = vi.hoisted(() => ({
  BehaviorOverride: {
    AUTO: 1,
    FORCE_OFF: 2,
    FORCE_PASSIVE: 3,
  },
  fetchIntegrationsStatus: vi.fn(),
  updateBehaviorOverride: vi.fn(),
}));

vi.mock("../../api/integrations", () => integrationsApi);

function Wrapper({ children }: { children: ReactNode }) {
  return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>;
}

describe("useIntegrationStatus", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads status and applies override updates", async () => {
    integrationsApi.fetchIntegrationsStatus.mockResolvedValueOnce({
      activeMode: BehaviorMode.PASSIVE,
      reason: "ready",
    });
    integrationsApi.updateBehaviorOverride.mockResolvedValueOnce({
      activeMode: BehaviorMode.OFF,
      reason: "forced off",
    });

    const { result } = renderHook(() => useIntegrationStatus(), {
      wrapper: Wrapper,
    });

    await waitFor(() => {
      expect(result.current.status?.activeMode).toBe(BehaviorMode.PASSIVE);
    });
    await act(async () => {
      await result.current.setOverride(BehaviorOverride.FORCE_OFF);
    });

    expect(result.current.status?.reason).toBe("forced off");
    expect(integrationsApi.updateBehaviorOverride).toHaveBeenCalledWith(BehaviorOverride.FORCE_OFF);
  });

  it("surfaces refresh errors without leaving loading stuck", async () => {
    integrationsApi.fetchIntegrationsStatus.mockRejectedValueOnce(new Error("registry down"));

    const { result } = renderHook(() => useIntegrationStatus(), {
      wrapper: Wrapper,
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.error).toContain("registry down");
  });
});
