import { createClient } from "@connectrpc/connect";
import {
  RunsService,
  RunStatus,
  type Run,
} from "@vrooli/proto-types/vrooli-bridge/v1/runs/runs_pb";
import {
  RunEventKind,
  type RunEvent,
} from "@vrooli/proto-types/vrooli-bridge/v1/channel/channel_pb";

import { transport } from "./client";

/**
 * Typed client for the RunsService — durable remote-execution history
 * (runs domain, OT-P0-005). GetRun returns a run plus its full persisted
 * event history (log/status/exit/artifact-ref); ListRuns is the newest-first
 * feed; AbortRun stops an in-flight run; StreamRunEvents is the live overlay.
 * The dashboard renders persisted history for terminal runs and can stream
 * live output for in-flight ones.
 */
export const runsClient = createClient(RunsService, transport);

export { RunStatus, RunEventKind };
export type { Run, RunEvent };
