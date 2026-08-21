/**
 * OverviewPanel tests — the one-glance summary composing tunnel, exposure, and
 * recovery. Each domain is mocked independently so a failure in one card does
 * not blank the others.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Mode } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { renderWithProviders } from "../../test-utils";
import {
  makeConfigMocks,
  makeConfigReadiness,
  makeConfigResponse,
  makeDriftResponse,
  makeTunnelConfig,
} from "../../test-utils/mocks/config";
import { makeTunnelMocks, makeTunnelStatus } from "../../test-utils/mocks/tunnel";
import { makeExposureMocks, makeExposure, makeLeasedExposure } from "../../test-utils/mocks/exposure";
import { makeRecoveryMocks, makeRecoveryState } from "../../test-utils/mocks/recovery";
import { makeProbesMocks } from "../../test-utils/mocks/probes";

vi.mock("../../api/config", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/config")>();
  return { ...actual, ...makeConfigMocks() };
});
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
vi.mock("../../api/probes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/probes")>();
  return { ...actual, ...makeProbesMocks() };
});

import { OverviewPanel } from "./OverviewPanel";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { i18n, setLocale } from "../../i18n";

describe("OverviewPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows readiness, tunnel status, the core/leased split, and recovery status", async () => {
    const { exposureClient } = await import("../../api/exposure");
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        config: makeTunnelConfig({ mode: 2 }),
        readiness: makeConfigReadiness({ syncReady: true, remoteAvailable: false, missingFields: [] }),
      }),
    );
    vi.mocked(exposureClient.listExposures).mockResolvedValueOnce({
      exposures: [makeExposure(), makeExposure({ scenario: "swarm-manager" }), makeLeasedExposure()],
    } as never);

    renderWithProviders(<OverviewPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.readinessStatus)).toHaveTextContent("Sync ready");
      expect(screen.getByTestId(selectors.overview.tunnelStatus)).toHaveTextContent(i18n.t(strings.overview.tunnelStatus.healthy));
    });
    expect(screen.getByTestId(selectors.overview.modeBadge)).toHaveTextContent("Local");
    expect(screen.getByTestId(selectors.overview.tunnelScore)).toHaveTextContent("100");
    expect(screen.getByTestId(selectors.overview.coreCount)).toHaveTextContent("2");
    expect(screen.getByTestId(selectors.overview.leasedCount)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.overview.recoveryStatus)).toHaveTextContent("Monitoring");
    expect(screen.getAllByTestId(selectors.overview.mobileRoute)).toHaveLength(3);
  });

  it("shows missing Cloudflare fields when remote setup is incomplete", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        readiness: makeConfigReadiness({
          syncReady: false,
          missingFields: ["CLOUDFLARE_API_TOKEN"],
        }),
      }),
    );

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.readinessStatus)).toHaveTextContent("Setup required");
    });
    expect(screen.getByTestId(selectors.overview.missingFields)).toHaveTextContent("CLOUDFLARE_API_TOKEN");
  });

  it("surfaces capability-scoped failures when remote drift or Access evidence cannot load", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getDrift).mockRejectedValueOnce(new Error("missing Cloudflare token"));
    vi.mocked(configClient.getAccessStatus).mockRejectedValueOnce(new Error("Access unavailable"));

    renderWithProviders(<OverviewPanel />);

    await waitFor(() => {
      expect(screen.getAllByText(i18n.t(strings.overview.driftUnavailable)).length).toBeGreaterThan(0);
      expect(screen.getByText(i18n.t(strings.overview.accessUnavailable))).toBeInTheDocument();
    });
    expect(screen.getByText(i18n.t(strings.overview.accessUnavailable)).closest("a")).toHaveAttribute("href", "/drift");
  });

  it("surfaces unavailable route classification instead of a healthy-looking zero", async () => {
    const { probesClient } = await import("../../api/probes");
    vi.mocked(probesClient.classify).mockRejectedValueOnce(new Error("classification unavailable"));

    renderWithProviders(<OverviewPanel />);

    await waitFor(() => {
      expect(screen.getByText(i18n.t(strings.overview.classificationUnavailable))).toBeInTheDocument();
    });
  });

  it("does not render an empty topology when the exposure inventory fails", async () => {
    const { exposureClient } = await import("../../api/exposure");
    vi.mocked(exposureClient.listExposures).mockRejectedValueOnce(new Error("inventory unavailable"));

    renderWithProviders(<OverviewPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.constellationError)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.overview.constellationRetry)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.overview.constellation)).not.toHaveTextContent("0 URL");
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
      expect(screen.getByTestId(selectors.overview.tunnelStatus)).toHaveTextContent(i18n.t(strings.overview.tunnelStatus.degraded));
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

  it("shows the remote tally when credentials are configured even if the persisted mode is the local default", async () => {
    // Regression for the reported bug: a tunnel set up via the Cloudflare
    // dashboard with credentials configured but the toggle never flipped off
    // the local default. The backend now reports the EFFECTIVE mode (remote),
    // so the operator sees the real managed/drift tally — not a misleading
    // "keys configured but not in use" callout.
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        config: makeTunnelConfig({ mode: Mode.LOCAL }),
        readiness: makeConfigReadiness({ desiredMode: Mode.REMOTE, remoteAvailable: true, missingFields: [] }),
      }),
    );
    vi.mocked(configClient.getDrift).mockResolvedValueOnce(
      makeDriftResponse({
        counts: { managed: 7, missing: 0, externalOk: 0, orphaned: 0, ignored: 0, unmanaged: 11 },
      }),
    );

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.modeCallout)).toHaveTextContent(
        "Mode: remote — 7 managed · 0 external · 11 drift",
      );
    });
    expect(screen.getByTestId(selectors.overview.modeCallout)).not.toHaveTextContent("not in use");
  });

  it("reads 'remote — N managed · M external · K drift' in remote mode", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        config: makeTunnelConfig({ mode: Mode.REMOTE }),
        readiness: makeConfigReadiness({ desiredMode: Mode.REMOTE, remoteAvailable: true, missingFields: [] }),
      }),
    );
    vi.mocked(configClient.getDrift).mockResolvedValueOnce(
      makeDriftResponse({
        counts: { managed: 3, missing: 0, externalOk: 2, orphaned: 0, ignored: 0, unmanaged: 1 },
      }),
    );

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.modeCallout)).toHaveTextContent(
        "Mode: remote — 3 managed · 2 external · 1 drift",
      );
    });
    expect(screen.getByTestId(selectors.overview.driftManaged)).toHaveTextContent("3");
    expect(screen.getByTestId(selectors.overview.driftUnmanaged)).toHaveTextContent("1");
  });

  it("reads plain local copy when local mode has no credentials", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        config: makeTunnelConfig({ mode: Mode.LOCAL }),
        readiness: makeConfigReadiness({ remoteAvailable: false }),
      }),
    );

    renderWithProviders(<OverviewPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.overview.modeCallout)).toHaveTextContent(
        "Mode: local — ingress is managed from the local cloudflared config",
      );
    });
  });
});
