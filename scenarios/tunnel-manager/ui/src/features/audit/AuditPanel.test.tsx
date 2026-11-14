/**
 * AuditPanel tests — port-compliance findings table and the violation summary.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeAuditMocks, makeAuditResult } from "../../test-utils/mocks/audit";

vi.mock("../../api/audit", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/audit")>();
  return { ...actual, ...makeAuditMocks() };
});

import { AuditPanel } from "./AuditPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("AuditPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when there are no routes", async () => {
    renderWithProviders(<AuditPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.empty)).toBeInTheDocument();
    });
  });

  it("renders findings and a zero-violation summary when all compliant", async () => {
    const { auditClient } = await import("../../api/audit");
    vi.mocked(auditClient.runAudit).mockResolvedValueOnce({
      results: [makeAuditResult()],
      violationCount: 0,
    } as never);

    renderWithProviders(<AuditPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.audit.table)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.audit.summary)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.audit.violationCount)).toHaveTextContent("No violations");
    expect(screen.getByTestId(selectors.audit.statusBadge)).toHaveTextContent("Compliant");
    expect(screen.getByTestId(selectors.audit.remediation)).toHaveTextContent("No action required");
  });

  it("surfaces violations with their status and count", async () => {
    const { auditClient } = await import("../../api/audit");
    vi.mocked(auditClient.runAudit).mockResolvedValueOnce({
      results: [
        makeAuditResult(),
        makeAuditResult({ subdomain: "drift", scenario: "drift", actualPort: 9999, status: 2, detail: "port drift" }),
      ],
      violationCount: 1,
    } as never);

    renderWithProviders(<AuditPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.audit.violationCount)).toHaveTextContent("1 violation");
    });
    expect(screen.getAllByTestId(selectors.audit.row)).toHaveLength(2);
    expect(
      screen.getAllByTestId(selectors.audit.statusBadge).some((el) => el.textContent.includes("Mismatch")),
    ).toBe(true);
  });

  it("filters findings by actionable violations", async () => {
    const { auditClient } = await import("../../api/audit");
    vi.mocked(auditClient.runAudit).mockResolvedValueOnce({
      results: [
        makeAuditResult(),
        makeAuditResult({ subdomain: "drift", scenario: "drift", actualPort: 9999, status: 2, detail: "port drift" }),
      ],
      violationCount: 1,
    } as never);
    const user = userEvent.setup();

    renderWithProviders(<AuditPanel />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.audit.row)).toHaveLength(2);
    });

    await user.selectOptions(screen.getByTestId(selectors.audit.statusFilter), "violations");
    expect(screen.getAllByTestId(selectors.audit.row)).toHaveLength(1);
    expect(screen.getByTestId(selectors.audit.remediation)).toHaveTextContent("reconcile exposure");
  });

  it("covers each typed compliance state and filter", async () => {
    const { auditClient } = await import("../../api/audit");
    const user = userEvent.setup();
    vi.mocked(auditClient.runAudit).mockResolvedValueOnce({
      results: [
        makeAuditResult({ subdomain: "missing-scenario", status: 3 }),
        makeAuditResult({ subdomain: "missing-port", status: 4 }),
        makeAuditResult({ subdomain: "unknown", status: 99 }),
      ],
      violationCount: 3,
    } as never);

    renderWithProviders(<AuditPanel />);
    await waitFor(() => expect(screen.getAllByTestId(selectors.audit.row)).toHaveLength(3));
    const filter = screen.getByTestId(selectors.audit.statusFilter);
    for (const value of ["compliant", "mismatch", "missing-scenario", "missing-port"]) {
      await user.selectOptions(filter, value);
      expect(screen.queryAllByTestId(selectors.audit.row).length).toBeLessThanOrEqual(1);
    }
    await user.selectOptions(filter, "all");
    expect(screen.getAllByTestId(selectors.audit.statusBadge)).toHaveLength(3);
  });

  it("re-runs the audit from the run button", async () => {
    const { auditClient } = await import("../../api/audit");
    const user = userEvent.setup();
    renderWithProviders(<AuditPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.audit.runButton)).toBeInTheDocument());

    const before = vi.mocked(auditClient.runAudit).mock.calls.length;
    await user.click(screen.getByTestId(selectors.audit.runButton));
    await waitFor(() => {
      expect(vi.mocked(auditClient.runAudit).mock.calls.length).toBeGreaterThan(before);
    });
  });

  it("shows the error state when runAudit rejects", async () => {
    const { auditClient } = await import("../../api/audit");
    vi.mocked(auditClient.runAudit).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<AuditPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.error)).toBeInTheDocument();
    });
  });
});
