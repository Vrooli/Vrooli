import { cleanup, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { expectNoA11yViolations, renderWithProviders } from "./test-utils";

const api = vi.hoisted(() => ({
  status: vi.fn(),
  items: vi.fn(),
  audit: vi.fn(),
  enrollmentStatus: vi.fn(),
  history: vi.fn(),
  accessRequests: vi.fn(),
  grants: vi.fn(),
  effectiveAccess: vi.fn(),
  revokeGrant: vi.fn(),
  assurance: vi.fn(),
  sources: vi.fn(),
  recovery: vi.fn(),
  preview: vi.fn(),
  generate: vi.fn(),
  reveal: vi.fn(),
  create: vi.fn(),
  createExport: vi.fn(),
  redeemExport: vi.fn(),
  lock: vi.fn(),
  unlock: vi.fn(),
  trash: vi.fn(),
  restore: vi.fn(),
  update: vi.fn(),
  createGrant: vi.fn(),
  approve: vi.fn(),
  deny: vi.fn(),
  createSource: vi.fn(),
  commit: vi.fn(),
  enroll: vi.fn(),
  setToken: vi.fn()
  ,exportAudit: vi.fn()
  ,sourceHealth: vi.fn()
}));

vi.mock("./lib/passwordManagerApi", () => ({
  createGrant: api.createGrant,
  completeEnrollment: api.enroll,
  createVaultItem: api.create,
  createVaultExport: api.createExport,
  redeemVaultExport: api.redeemExport,
  approveAccessRequest: api.approve,
  generatePassword: api.generate,
  getVaultStatus: api.status,
  listVaultItems: api.items,
  listAccessRequests: api.accessRequests,
  listVaultItemHistory: api.history,
  listGrants: api.grants,
  getGrantEffectiveAccess: api.effectiveAccess,
  revokeGrant: api.revokeGrant,
  issueAssurance: api.assurance,
  lockVault: api.lock,
  revealVaultItem: api.reveal,
  setConfiguredOwnerToken: api.setToken,
  denyAccessRequest: api.deny,
  createSource: api.createSource,
  commitVaultImport: api.commit,
  getRecoveryStatus: api.recovery,
  getEnrollmentStatus: api.enrollmentStatus,
  listSources: api.sources,
  previewVaultImport: api.preview,
  restoreVaultItem: api.restore,
  trashVaultItem: api.trash,
  updateVaultItem: api.update,
  unlockVault: api.unlock,
  getAudit: api.audit
  ,getSourceHealth: api.sourceHealth
  ,exportAudit: api.exportAudit
}));

describe("Secrets Manager application accessibility", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the initial vault without axe violations", async () => {
    api.status.mockResolvedValue({ vault_id: "personal", status: "locked", key_available: true, supports_recovery: true });
    api.items.mockResolvedValue({ items: [] });
    api.audit.mockResolvedValue({ events: [] });
    api.enrollmentStatus.mockResolvedValue({ bootstrap_configured: false, enrolled: true });
    api.history.mockResolvedValue({ history: [] });

    const { container } = renderWithProviders(<App />);
    await waitFor(() => expect(container.querySelector(".empty-state")).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
