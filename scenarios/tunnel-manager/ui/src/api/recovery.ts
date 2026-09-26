import { createClient } from "@connectrpc/connect";
import {
  RecoveryService,
  RecoveryStatus,
  EventOutcome,
  type RecoveryState,
  type RecoveryEvent,
} from "@vrooli/proto-types/tunnel-manager/v1/recovery/recovery_pb";

import { transport } from "./client";

// recoveryClient is the generated Connect-Web client for RecoveryService —
// the live auto-recovery state machine + event log. Backs the Recovery &
// Events surface under ui/src/features/recovery/.
export const recoveryClient = createClient(RecoveryService, transport);

/** getState returns the live recovery state-machine snapshot. */
export async function getState(): Promise<RecoveryState | undefined> {
  const resp = await recoveryClient.getState({});
  return resp.state;
}

/** listEvents returns the recovery event log, newest first. */
export async function listEvents(limit = 0): Promise<RecoveryEvent[]> {
  const resp = await recoveryClient.listEvents({ limit });
  return resp.events;
}

/** recover triggers a manual recovery attempt (force bypasses the breaker). */
export async function recover(force = false): Promise<{ outcome: EventOutcome; event?: RecoveryEvent }> {
  const resp = await recoveryClient.recover({ force });
  return { outcome: resp.outcome, event: resp.event };
}

export { RecoveryStatus, EventOutcome };
export type { RecoveryState, RecoveryEvent };
