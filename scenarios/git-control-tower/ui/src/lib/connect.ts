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
import { EvidenceService } from "@vrooli/proto-types/git-control-tower/v1/evidence/evidence_pb";

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

/** Shared phase-agnostic run and evidence surface for every review tab. */
export const evidenceClient: Client<typeof EvidenceService> = createClient(
  EvidenceService,
  transport,
);
