/**
 * Mock builders for `api/audit`. See `test-utils/mocks/api.ts` for the
 * builder/hoisting rationale.
 */
import { vi } from "vitest";

export interface AuditMocks {
  auditClient: {
    runAudit: ReturnType<typeof vi.fn>;
  };
}

export const makeAuditResult = (overrides: Record<string, unknown> = {}) => ({
  subdomain: "agent-manager",
  scenario: "agent-manager",
  expectedPort: 21001,
  actualPort: 21001,
  status: 1, // COMPLIANT
  detail: "",
  ...overrides,
});

export const makeAuditMocks = (): AuditMocks => ({
  auditClient: {
    runAudit: vi.fn().mockResolvedValue({ results: [], violationCount: 0 }),
  },
});
