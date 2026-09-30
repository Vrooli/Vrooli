// ============================================================================
// Connect-RPC client wiring (Plan B §4.1)
// ============================================================================
//
// GCT's UI talks to proto+Connect services through a single-origin transport.
// The REST clients in api-*.ts use resolveApiBase({ appendSuffix: true }) (the
// "/api/v1" surface); Connect procedures live at the bare origin under their
// generated package path (e.g. "/vrooli.git_control_tower.v1.baselines.
// BaselinesService/ListBaselines"), so the transport uses the un-suffixed base.
//
// Cross-scenario Test Genie reads are NEVER made directly from the browser.
// They flow through GCT's phase-agnostic EvidenceService on this same origin,
// keeping the UI single-origin and artifact paths private.

import { createClient, type Client } from "@connectrpc/connect";
import { resolveApiBase, createScenarioConnectTransport } from "@vrooli/api-base";
import { BaselinesService } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import { AuthService } from "@vrooli/proto-types/git-control-tower/v1/auth/auth_pb";
import { BranchService } from "@vrooli/proto-types/git-control-tower/v1/branch/branch_pb";
import { AuditorService } from "@vrooli/proto-types/git-control-tower/v1/auditor/auditor_pb";
import { EvidenceService } from "@vrooli/proto-types/git-control-tower/v1/evidence/evidence_pb";
import { HumanControlService } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import { RepoService } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { ReviewService } from "@vrooli/proto-types/git-control-tower/v1/review/review_pb";

// Bare origin (no "/api/v1" suffix): Connect appends the full procedure path.
export const transport = createScenarioConnectTransport({ baseUrl: resolveApiBase() });

/**
 * Connect client for GCT's cross-surface baseline substrate. Call methods
 * (`baselinesClient.listBaselines(...)`) through React Query hooks rather than
 * ad-hoc wrappers — the generated client owns proto encoding, error parsing,
 * and cancellation.
 */
export const baselinesClient: Client<typeof BaselinesService> = createClient(
  BaselinesService,
  transport,
);

/** Same-origin sign-in facade; the API forwards credentials to the IdP. */
export const authClient: Client<typeof AuthService> = createClient(AuthService, transport);

/** Typed repository branch client. */
export const branchClient: Client<typeof BranchService> = createClient(BranchService, transport);

/** Typed auditor preview and exact-intent fix client. */
export const auditorClient: Client<typeof AuditorService> = createClient(AuditorService, transport);

/** Shared phase-agnostic run and evidence surface for every review tab. */
export const evidenceClient: Client<typeof EvidenceService> = createClient(
  EvidenceService,
  transport,
);

/** Typed authority, preview, and single-use mutation-intent client. */
export const humanControlClient: Client<typeof HumanControlService> = createClient(
  HumanControlService,
  transport,
);

/** Typed repository command client. */
export const repoClient: Client<typeof RepoService> = createClient(RepoService, transport);

/** Typed advisory review lifecycle client. */
export const reviewClient: Client<typeof ReviewService> = createClient(ReviewService, transport);
