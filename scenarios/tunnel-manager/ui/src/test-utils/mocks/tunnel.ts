/**
 * Mock builders for `api/tunnel`. See `test-utils/mocks/api.ts` for the
 * builder/hoisting rationale.
 */
import { vi } from "vitest";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

export interface TunnelMocks {
  tunnelClient: {
    getStatus: ReturnType<typeof vi.fn>;
    listMetrics: ReturnType<typeof vi.fn>;
    scrape: ReturnType<typeof vi.fn>;
  };
}

export const makeTunnelStatus = (overrides: Record<string, unknown> = {}) => ({
  status: "healthy",
  systemd: "active",
  ready: "ok",
  readyLatencyMs: 12,
  score: 100,
  message: "",
  checkedAt: timestampFromDate(new Date(2026, 5, 18, 12, 0, 0)),
  ...overrides,
});

export const makeMetricsSample = (overrides: Record<string, unknown> = {}) => ({
  id: "sample-1",
  haConnections: 4,
  requestErrors: 0,
  activeStreams: 2,
  smoothedRttMs: 18.5,
  scrapedAt: timestampFromDate(new Date(2026, 5, 18, 12, 0, 0)),
  ...overrides,
});

export const makeTunnelMocks = (): TunnelMocks => ({
  tunnelClient: {
    getStatus: vi.fn().mockResolvedValue({ status: makeTunnelStatus(), latestMetrics: makeMetricsSample() }),
    listMetrics: vi.fn().mockResolvedValue({ samples: [] }),
    scrape: vi.fn().mockResolvedValue({ sample: makeMetricsSample() }),
  },
});
