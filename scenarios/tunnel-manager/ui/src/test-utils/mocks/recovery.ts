/**
 * Mock builders for `api/recovery`. See `test-utils/mocks/api.ts` for the
 * builder/hoisting rationale.
 */
import { vi } from "vitest";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

export interface RecoveryMocks {
  recoveryClient: {
    getState: ReturnType<typeof vi.fn>;
    listEvents: ReturnType<typeof vi.fn>;
    recover: ReturnType<typeof vi.fn>;
  };
}

export const makeRecoveryState = (overrides: Record<string, unknown> = {}) => ({
  status: 2, // MONITORING
  consecFailures: 0,
  backoffLevel: 0,
  failedRecoveries: 0,
  circuitOpen: false,
  lastCheck: timestampFromDate(new Date(2026, 5, 18, 12, 0, 0)),
  lastRecovery: undefined,
  nextRetryAfter: undefined,
  ...overrides,
});

export const makeRecoveryEvent = (overrides: Record<string, unknown> = {}) => ({
  id: "event-1",
  trigger: "ready_failure",
  action: "systemctl_restart",
  outcome: 1, // SUCCESS
  details: "restarted cloudflared",
  attempt: 1,
  createdAt: timestampFromDate(new Date(2026, 5, 18, 11, 0, 0)),
  ...overrides,
});

export const makeRecoveryMocks = (): RecoveryMocks => ({
  recoveryClient: {
    getState: vi.fn().mockResolvedValue({ state: makeRecoveryState() }),
    listEvents: vi.fn().mockResolvedValue({ events: [] }),
    recover: vi.fn().mockResolvedValue({ outcome: 1, event: makeRecoveryEvent() }),
  },
});
