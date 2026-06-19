/**
 * OverviewPanel tests — the one-glance summary composing tunnel, exposure, and
 * recovery. Each domain is mocked independently so a failure in one card does
 * not blank the others.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeTunnelMocks, makeTunnelStatus } from "../../test-utils/mocks/tunnel";
import { makeExposureMocks, makeExposure, makeLeasedExposure } from "../../test-utils/mocks/exposure";
import { makeRecoveryMocks, makeRecoveryState } from "../../test-utils/mocks/recovery";

vi.mock("../../api/tunnel", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/tunnel")>();
  return { ...actual, ...makeTunnelMocks() };
});
vi.mock("../../api/exposure", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/exposure")>();
  return { ...actual, ...makeExposureMocks() };
});
vi.mock("../../api/recovery", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/recovery")>();
  return { ...actual, ...makeRecoveryMocks() };
});

import { OverviewPanel } from "./OverviewPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("OverviewPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows tunnel status, the core/leased split, and recovery status", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listExposures).mockResolvedValueOnce({
      exposures: [makeExposure(), makeExposure({ scenario: "swarm-manager" }), makeLeasedExposure()],
    } as never);

    renderWithProviders(<OverviewPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.tunnelStatus)).toHaveTextContent("healthy");
    });
    expect(screen.getByTestId(selectors.overview.tunnelScore)).toHaveTextContent("100");
    expect(screen.getByTestId(selectors.overview.coreCount)).toHaveTextContent("2");
    expect(screen.getByTestId(selectors.overview.leasedCount)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.overview.recoveryStatus)).toHaveTextContent("Monitoring");
  });

  it("warns when the recovery circuit breaker is open", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    vi.mocked(recoveryClient.getState).mockResolvedValueOnce({
      state: makeRecoveryState({ status: 4, circuitOpen: true }),
    } as never);

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.circuitWarning)).toBeInTheDocument();
    });
  });

  it("renders a degraded tunnel badge without crashing the other cards", async () => {
    const { tunnelClient } = await import("../../api/tunnel");
    vi.mocked(tunnelClient.getStatus).mockResolvedValueOnce({
      status: makeTunnelStatus({ status: "degraded", score: 60 }),
    } as never);

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.tunnelStatus)).toHaveTextContent("degraded");
    });
    expect(screen.getByTestId(selectors.overview.recoveryCard)).toBeInTheDocument();
  });

  it("shows the tunnel error state when getStatus rejects", async () => {
    const { tunnelClient } = await import("../../api/tunnel");
    vi.mocked(tunnelClient.getStatus).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.queryState.error).length).toBeGreaterThan(0);
    });
  });
});
