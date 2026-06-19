/**
 * MetricsPanel tests — tunnel metrics time-series plus the per-route probe
 * history and classification, with the scrape / run-probes actions.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeTunnelMocks, makeMetricsSample } from "../../test-utils/mocks/tunnel";
import { makeProbesMocks, makeProbeResult, makeRouteClassification } from "../../test-utils/mocks/probes";

vi.mock("../../api/tunnel", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/tunnel")>();
  return { ...actual, ...makeTunnelMocks() };
});
vi.mock("../../api/probes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/probes")>();
  return { ...actual, ...makeProbesMocks() };
});

import { MetricsPanel } from "./MetricsPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("MetricsPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders empty states for metrics and probes by default", async () => {
    renderWithProviders(<MetricsPanel />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.queryState.empty).length).toBeGreaterThanOrEqual(2);
    });
  });

  it("renders metrics samples when present", async () => {
    const { tunnelClient } = await import("../../api/tunnel");
    vi.mocked(tunnelClient.listMetrics).mockResolvedValueOnce({
      samples: [makeMetricsSample(), makeMetricsSample({ id: "sample-2", haConnections: 3 })],
    } as never);

    renderWithProviders(<MetricsPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.metrics.table)).toBeInTheDocument();
    });
    expect(screen.getAllByTestId(selectors.metrics.row)).toHaveLength(2);
  });

  it("renders probe history and the route classification", async () => {
    const { probesClient } = await import("../../api/probes");
    vi.mocked(probesClient.listProbes).mockResolvedValueOnce({
      results: [makeProbeResult(), makeProbeResult({ id: "probe-2", kind: 2, status: 2 })],
    } as never);
    vi.mocked(probesClient.classify).mockResolvedValueOnce({
      classifications: [makeRouteClassification({ classification: 3 })],
    } as never);

    renderWithProviders(<MetricsPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.metrics.probesTable)).toBeInTheDocument();
    });
    expect(screen.getAllByTestId(selectors.metrics.probesRow)).toHaveLength(2);
    expect(screen.getByTestId(selectors.metrics.classification)).toBeInTheDocument();
  });

  it("triggers a scrape and a probe run from the action buttons", async () => {
    const { tunnelClient } = await import("../../api/tunnel");
    const { probesClient } = await import("../../api/probes");
    const user = userEvent.setup();
    renderWithProviders(<MetricsPanel />);

    await user.click(screen.getByTestId(selectors.metrics.scrapeButton));
    await waitFor(() => expect(tunnelClient.scrape).toHaveBeenCalledTimes(1));

    await user.click(screen.getByTestId(selectors.metrics.runProbesButton));
    await waitFor(() => expect(probesClient.runProbes).toHaveBeenCalledTimes(1));
  });

  it("shows the probe error state when listProbes rejects", async () => {
    const { probesClient } = await import("../../api/probes");
    vi.mocked(probesClient.listProbes).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<MetricsPanel />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.queryState.error).length).toBeGreaterThanOrEqual(1);
    });
  });
});
