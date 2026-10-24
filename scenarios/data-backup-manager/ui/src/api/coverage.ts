/**
 * Coverage domain — UI ↔ API boundary over CoverageService. Coverage is the
 * first-real-backup readiness surface: it composes discovery suggestions with
 * the targets/plans/runs/restores catalogs into one report (what is registered,
 * recommended-but-unregistered, sensitive, planned, backed-up, verified) plus a
 * bulk default-acceptance action.
 *
 * `getCoverageReport` is read-only and derives everything live. `acceptDefaults`
 * bulk-registers non-sensitive discovered durable targets; sensitive
 * credential/token suggestions are skipped unless `includeSensitive` is set, and
 * `dryRun` registers nothing.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { CoverageService } from "@vrooli/proto-types/data-backup-manager/v1/coverage/coverage_pb";
import type {
  CoverageReport,
  AcceptDefaultTargetsResponse,
} from "@vrooli/proto-types/data-backup-manager/v1/coverage/coverage_pb";

import { transport } from "./client";

export const coverageClient: Client<typeof CoverageService> = createClient(
  CoverageService,
  transport,
);

export async function getCoverageReport(): Promise<CoverageReport | undefined> {
  const res = await coverageClient.getCoverageReport({});
  return res.report;
}

export interface AcceptDefaultsInput {
  includeSensitive?: boolean;
  dryRun?: boolean;
}

export async function acceptDefaultTargets(
  input: AcceptDefaultsInput = {},
): Promise<AcceptDefaultTargetsResponse> {
  return coverageClient.acceptDefaultTargets({
    includeSensitive: input.includeSensitive ?? false,
    dryRun: input.dryRun ?? false,
  });
}

export type { CoverageReport, AcceptDefaultTargetsResponse };
