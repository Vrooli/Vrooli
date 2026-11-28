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
import { ProposalService } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { RepoService } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { ReviewService } from "@vrooli/proto-types/git-control-tower/v1/review/review_pb";
import { ScenarioService } from "@vrooli/proto-types/git-control-tower/v1/scenarios/scenarios_pb";
import { AdvisoryService } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { SourceDistributionService } from "@vrooli/proto-types/git-control-tower/v1/source_distributions/source_distributions_pb";
import { TidinessService } from "@vrooli/proto-types/git-control-tower/v1/tidiness/tidiness_pb";
import { AgentReadService } from "@vrooli/proto-types/git-control-tower/v1/agent_reads/agent_reads_pb";
import { IntegrationService } from "@vrooli/proto-types/git-control-tower/v1/integration/integration_pb";
import { VisualCaptureService } from "@vrooli/proto-types/git-control-tower/v1/visual_captures/visual_captures_pb";

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

/** Typed commit-proposal client: drafts, freshness, and human-only apply. */
export const proposalClient: Client<typeof ProposalService> = createClient(ProposalService, transport);

/** Typed repository command client. */
export const repoClient: Client<typeof RepoService> = createClient(RepoService, transport);

/** Typed advisory review lifecycle client. */
export const reviewClient: Client<typeof ReviewService> = createClient(ReviewService, transport);

/** Typed scenario discovery and agent-envelope reads. */
export const scenarioClient: Client<typeof ScenarioService> = createClient(ScenarioService, transport);

/** Evidence collection and explicitly requested message-only drafting. */
export const advisoryClient: Client<typeof AdvisoryService> = createClient(AdvisoryService, transport);

/** Read-only projection of source-ramp distribution evidence. */
export const sourceDistributionClient: Client<typeof SourceDistributionService> = createClient(SourceDistributionService, transport);

/** Typed projection over tidiness-manager-owned scores, issues and scans. */
export const tidinessClient: Client<typeof TidinessService> = createClient(TidinessService, transport);

/** Verified, request-scoped projection over Agent Manager-owned run reads. */
export const agentReadClient: Client<typeof AgentReadService> = createClient(AgentReadService, transport);

/** GCT-owned integration capability and host-neutral collaboration status reads. */
export const integrationClient: Client<typeof IntegrationService> = createClient(IntegrationService, transport);

/** Typed metadata/path reads for GCT-owned visual capture storage. */
export const visualCaptureClient: Client<typeof VisualCaptureService> = createClient(VisualCaptureService, transport);

// Same-origin intake only; UI Health owns retained browser layout evidence.
import { VisualHealthService } from "@vrooli/proto-types/ui-health/v1/visualhealth/visualhealth_pb";
export const browserObservationClient = createClient(VisualHealthService, transport);
