import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { AuditStatus, type Audit } from "../../api/audits";
import { AuditReport } from "./AuditReport";

afterEach(() => cleanup());

function makeInventory(over: Partial<Record<string, unknown>> = {}) {
  return {
    files: 5n,
    directories: 2n,
    symlinks: 0n,
    other: 0n,
    regularBytes: 1024n,
    pathListSha256: "abc123abc123abc123",
    treeContentSha256: "def456def456def456",
    sqlite: [],
    unreadablePaths: [],
    ...over,
  };
}

function makeAudit(over: Partial<Record<string, unknown>> = {}): Audit {
  return {
    id: "a-1",
    targetId: "t-1",
    destinationId: "d-1",
    snapshotId: "snap-1",
    status: AuditStatus.COMPLETED,
    includeContentHash: true,
    includeSqliteChecks: true,
    restorable: true,
    live: makeInventory(),
    snapshot: makeInventory(),
    comparison: { matches: true, mismatches: [], liveNewerThanSnapshot: false },
    error: "",
    ...over,
  } as unknown as Audit;
}

describe("AuditReport", () => {
  it("renders a loading state", () => {
    renderWithProviders(<AuditReport audit={undefined} loading error={false} />);
    expect(screen.getByTestId(selectors.audits.report)).toHaveTextContent(strings.audits.running);
  });

  it("renders an error state", () => {
    renderWithProviders(<AuditReport audit={undefined} loading={false} error />);
    expect(screen.getByTestId(selectors.audits.report)).toHaveTextContent(strings.audits.error);
  });

  it("renders a PASS verdict when inventories match", () => {
    renderWithProviders(<AuditReport audit={makeAudit()} loading={false} error={false} />);
    expect(screen.getByTestId(selectors.audits.verdict)).toHaveTextContent(strings.audits.verdictPass);
  });

  it("renders mismatches and a plain DIFF verdict (no drift)", () => {
    const audit = makeAudit({
      comparison: { matches: false, mismatches: ["file count: live=6 snapshot=5"], liveNewerThanSnapshot: false },
    });
    renderWithProviders(<AuditReport audit={audit} loading={false} error={false} />);
    expect(screen.getByTestId(selectors.audits.verdict)).toHaveTextContent(strings.audits.verdictDiff);
    expect(screen.getByTestId(selectors.audits.report)).toHaveTextContent("file count: live=6 snapshot=5");
  });

  it("explains a mismatch as drift when live is newer", () => {
    const audit = makeAudit({
      comparison: { matches: false, mismatches: ["path-list hash differs"], liveNewerThanSnapshot: true },
    });
    renderWithProviders(<AuditReport audit={audit} loading={false} error={false} />);
    expect(screen.getByTestId(selectors.audits.verdict)).toHaveTextContent(strings.audits.verdictDrift);
  });

  it("surfaces a FAILED verdict with the error and not-restorable", () => {
    const audit = makeAudit({
      status: AuditStatus.FAILED,
      restorable: false,
      live: undefined,
      snapshot: undefined,
      comparison: undefined,
      error: "snapshot restore: repo offline",
    });
    renderWithProviders(<AuditReport audit={audit} loading={false} error={false} />);
    expect(screen.getByTestId(selectors.audits.verdict)).toHaveTextContent(/repo offline/i);
    expect(screen.getByTestId(selectors.audits.report)).toHaveTextContent(strings.audits.notRestorable);
  });

  it("renders per-SQLite integrity facts", () => {
    const audit = makeAudit({
      snapshot: makeInventory({
        sqlite: [{ path: "events.db", integrityStatus: "ok", pageCount: 10n, pageSize: 4096n, schemaSha256: "s", tableCount: 3n }],
      }),
    });
    renderWithProviders(<AuditReport audit={audit} loading={false} error={false} />);
    const report = screen.getByTestId(selectors.audits.report);
    expect(report).toHaveTextContent(/events\.db/);
    expect(report).toHaveTextContent(/=ok/);
  });
});
