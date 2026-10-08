/**
 * Mock builders for `api/probes`. See `test-utils/mocks/api.ts` for the
 * builder/hoisting rationale.
 */
import { vi } from "vitest";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

export interface ProbesMocks {
  probesClient: {
    runProbes: ReturnType<typeof vi.fn>;
    listProbes: ReturnType<typeof vi.fn>;
    classify: ReturnType<typeof vi.fn>;
  };
}

export const makeProbeResult = (overrides: Record<string, unknown> = {}) => ({
  id: "probe-1",
  subdomain: "agent-manager",
  kind: 1, // INTERNAL
  status: 1, // UP
  latencyMs: 8,
  statusCode: 200,
  errorMsg: "",
  createdAt: timestampFromDate(new Date(2026, 5, 18, 12, 0, 0)),
  ...overrides,
});

export const makeRouteClassification = (overrides: Record<string, unknown> = {}) => ({
  subdomain: "agent-manager",
  classification: 1, // HEALTHY
  internal: 1,
  external: 1,
  assessment: "reachable",
  ...overrides,
});

export const makeProbesMocks = (): ProbesMocks => ({
  probesClient: {
    runProbes: vi.fn().mockResolvedValue({ results: [] }),
    listProbes: vi.fn().mockResolvedValue({ results: [] }),
    classify: vi.fn().mockResolvedValue({ classifications: [] }),
  },
});
