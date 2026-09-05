/**
 * Mock builders for `api/exposure`. Call `makeExposureMocks()` from inside a
 * `vi.mock("../../api/exposure", …)` factory closure (never at module top
 * level — see `test-utils/mocks/api.ts` for the hoisting rationale). The
 * `...actual` spread at the call site keeps re-exported enums/types intact.
 */
import { vi } from "vitest";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

export interface ExposureMocks {
  exposureClient: {
    listExposures: ReturnType<typeof vi.fn>;
    listLeases: ReturnType<typeof vi.fn>;
    expose: ReturnType<typeof vi.fn>;
    extendLease: ReturnType<typeof vi.fn>;
    revokeLease: ReturnType<typeof vi.fn>;
    isExposed: ReturnType<typeof vi.fn>;
    reconcile: ReturnType<typeof vi.fn>;
  };
}

/** A reconciled exposure row; override any field per test. */
export const makeExposure = (overrides: Record<string, unknown> = {}) => ({
  scenario: "agent-manager",
  subdomain: "agent-manager",
  publicUrl: "https://agent-manager.itsagitime.com",
  localPort: 21001,
  tier: "core",
  enabled: true,
  lease: undefined,
  ...overrides,
});

/** A leased exposure row, with a lease expiring in `days` days. */
export const makeLeasedExposure = (overrides: Record<string, unknown> = {}, days = 7) => {
  const expires = new Date(2026, 5, 25, 12, 0, 0);
  expires.setDate(expires.getDate() + days);
  return makeExposure({
    scenario: "image-tools",
    subdomain: "image-tools",
    publicUrl: "https://image-tools.itsagitime.com",
    localPort: 21240,
    tier: "leased",
    lease: {
      id: "lease-1",
      scenario: "image-tools",
      requestedBy: "operator",
      createdAt: timestampFromDate(new Date(2026, 5, 18, 12, 0, 0)),
      expiresAt: timestampFromDate(expires),
      extendedCount: 0,
      status: 1,
    },
    ...overrides,
  });
};

export const makeExposureMocks = (): ExposureMocks => ({
  exposureClient: {
    listExposures: vi.fn().mockResolvedValue({ exposures: [] }),
    listLeases: vi.fn().mockResolvedValue({ leases: [] }),
    expose: vi.fn().mockResolvedValue({ lease: undefined, publicUrl: "" }),
    extendLease: vi.fn().mockResolvedValue({ lease: undefined }),
    revokeLease: vi.fn().mockResolvedValue({ retracted: true }),
    isExposed: vi.fn().mockResolvedValue({ exposed: false, publicUrl: "" }),
    reconcile: vi.fn().mockResolvedValue({ coreEnsured: 0, leasesReaped: 0 }),
  },
});
