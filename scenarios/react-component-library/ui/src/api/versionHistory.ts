import type { Adoption } from "./adoptions";
import { adoptionsClient } from "./adoptions";
import { listComponentTestReports, type ComponentTestReport } from "./componentTests";
import { listVersionLedger, type VersionLedgerRow } from "./versionLedger";
import { versionsClient, type Version } from "./versions";

export interface VersionHistoryRow {
  version: Version;
  ledger?: VersionLedgerRow;
  testReport?: ComponentTestReport;
  adopters: Adoption[];
}

function reportTime(report: ComponentTestReport): number {
  const createdAt = report.createdAt;
  return createdAt ? createdAt.toDate().getTime() : 0;
}

/**
 * Join the four version-history sources once, before the table renders.
 * The row component receives an already-correlated view and never performs
 * one request per version.
 */
export async function listVersionHistory(componentId: string): Promise<VersionHistoryRow[]> {
  const [versionsResponse, ledger, reports, adoptionResponse] = await Promise.all([
    versionsClient.listVersions({ componentId, limit: 0 }),
    listVersionLedger(componentId),
    listComponentTestReports({ componentId, limit: 0 }),
    adoptionsClient.listAdoptions({ componentId, limit: 0 }),
  ]);

  const ledgerByVersion = new Map(ledger.map((row) => [row.version, row] as const));
  const latestReportByVersion = new Map<string, ComponentTestReport>();
  for (const report of reports) {
    const current = latestReportByVersion.get(report.rootVersion);
    if (!current || reportTime(report) >= reportTime(current)) {
      latestReportByVersion.set(report.rootVersion, report);
    }
  }
  const adoptionsByVersion = new Map<string, Adoption[]>();
  for (const adoption of adoptionResponse.adoptions) {
    const version = adoption.adoptedVersion;
    if (!version) continue;
    const current = adoptionsByVersion.get(version) ?? [];
    current.push(adoption);
    adoptionsByVersion.set(version, current);
  }

  return versionsResponse.versions.map((version) => ({
    version,
    ledger: ledgerByVersion.get(version.version),
    testReport: latestReportByVersion.get(version.version),
    adopters: adoptionsByVersion.get(version.version) ?? [],
  }));
}
