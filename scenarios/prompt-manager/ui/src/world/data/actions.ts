/**
 * Heartbeat actions available in the world HUD. Member conversation state and
 * sends live in conversations.ts and use the member-bound CreateRun operation.
 */
import { stopRunningAgent, triggerHeartbeat } from '@/services/heartbeatService'
import type { Signal } from '../sim'

export interface WorldActions {
  /** Trigger the member's heartbeat now. Returns the run id when one was created. */
  runNow(teamId: string, agentId: string): Promise<{ runId?: string; message: string }>
  /** Stop the member's active run. */
  stop(teamId: string, agentId: string): Promise<void>
  /** Clear a Failed state locally; the next run clears it too. */
  acknowledgeFailure(agentId: string): void
}

export function createWorldActions(dispatch: (signals: Signal[]) => void, now: () => number = () => Date.now() / 1000): WorldActions {
  return {
    runNow: async (teamId, agentId) => {
      const response = await triggerHeartbeat(teamId, agentId)
      return { runId: response.runId || undefined, message: response.status }
    },
    stop: async (teamId, agentId) => {
      await stopRunningAgent(teamId, agentId)
    },
    acknowledgeFailure: (agentId) => {
      dispatch([{ kind: 'failed.acknowledged', agentId, at: now() }])
    },
  }
}
