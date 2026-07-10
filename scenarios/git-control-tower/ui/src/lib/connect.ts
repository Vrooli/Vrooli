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
// Cross-scenario reads (test-genie runs for the Workflows tab) are NEVER made
// directly from the browser (Decision 3) — they flow through GCT's own
// WorkflowReplayService, which is mounted on this same origin. That keeps the
// UI single-origin (no CORS) and lets GCT apply scenario-context filtering
// before the browser sees any test-genie data.

import { createClient, type Client } from "@connectrpc/connect";
import { resolveApiBase, createScenarioConnectTransport } from "@vrooli/api-base";
import { BaselinesService } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import { WorkflowReplayService } from "@vrooli/proto-types/git-control-tower/v1/workflowreplay/workflowreplay_pb";

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

/**
 * Connect client for GCT's WorkflowReplayService — the single-origin proxy over
 * typed test-genie run evidence. The Workflows tab reads runs through
 * this; binary video bytes stream over the separate REST video-proxy route.
 */
export const workflowReplayClient: Client<typeof WorkflowReplayService> = createClient(
  WorkflowReplayService,
  transport,
);
