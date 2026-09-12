/**
 * MetricsStrip tests. The load-bearing behavior: when runs have measured
 * physical (deduped) bytes, the strip renders the dedup line; when no run
 * measured repo growth (physical 0), it renders the "not measured yet" line
 * instead of a misleading 0× ratio. The test i18n runs in key-path mode, so we
 * assert on the rendered string key (which branch), not translated copy.
 */
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

vi.mock("../../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/runs")>()),
  getRunStats: vi.fn(),
}));

import * as runsApi from "../../api/runs";
import { MetricsStrip } from "./MetricsStrip";

const baseStats = {
  totalRuns: 5n,
  completed: 5n,
  partialFailed: 0n,
  failed: 0n,
  successRate: 1,
  p50DurationMs: 1000n,
  p95DurationMs: 2000n,
  totalBytes: 2_000_000_000n,
  avgBytesPerRun: 400_000_000n,
  avgThroughputBytesPerSec: 1_000_000,
  window: 5n,
  totalPhysicalBytes: 100_000_000n,
  dedupRatio: 20,
};

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

it("renders the dedup line when physical bytes are measured", async () => {
  vi.mocked(runsApi.getRunStats).mockResolvedValue(baseStats as never);
  renderWithProviders(<MetricsStrip />);

  await waitFor(() => expect(screen.getByTestId(selectors.overview.metrics)).toBeInTheDocument());
  await waitFor(() => expect(screen.getByText(strings.overview.metricsDedup)).toBeInTheDocument());
  expect(screen.queryByText(strings.overview.metricsDedupEmpty)).not.toBeInTheDocument();
});

it("renders the not-measured line when no run reported physical bytes", async () => {
  vi.mocked(runsApi.getRunStats).mockResolvedValue({
    ...baseStats,
    totalPhysicalBytes: 0n,
    dedupRatio: 0,
  } as never);
  renderWithProviders(<MetricsStrip />);

  await waitFor(() => expect(screen.getByText(strings.overview.metricsDedupEmpty)).toBeInTheDocument());
  expect(screen.queryByText(strings.overview.metricsDedup)).not.toBeInTheDocument();
});
