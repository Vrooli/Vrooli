import { getActiveMicLeases, subscribeMicLeases, type MicLeaseSnapshot } from "./micOwnership";
import { selectStaleLeases, type VoiceStateLite } from "./micLifecyclePolicy";

export interface MicLeaseRecoveryOptions {
  getVoiceState: () => VoiceStateLite;
  isStarting: () => boolean;
  isPassiveListenerActive: () => boolean;
  onStaleState: (stale: boolean) => void;
  recover: (input: { voiceState: VoiceStateLite; passiveListenerActive: boolean; reason: "invariant-violation" }) => void;
  getLeases?: () => MicLeaseSnapshot[];
  subscribe?: (listener: (leases: MicLeaseSnapshot[]) => void) => () => void;
}

/** Keep registry truth and workflow state coordinated, including synchronous re-entrant releases. */
export function installMicLeaseRecovery(options: MicLeaseRecoveryOptions): () => void {
  let recoveryInProgress = false;
  const evaluate = (snapshots: MicLeaseSnapshot[]) => {
    // A start can acquire its lease before React commits "preparing".
    const voiceState = options.isStarting() ? "preparing" : options.getVoiceState();
    const passiveListenerActive = options.isPassiveListenerActive();
    const stale = selectStaleLeases({
      leases: snapshots.map(({ id, owner }) => ({ id, owner })),
      voiceState,
      passiveListenerActive,
    }).length > 0;
    options.onStaleState(stale);
    if (!stale || recoveryInProgress) return;

    recoveryInProgress = true;
    try {
      options.recover({ voiceState, passiveListenerActive, reason: "invariant-violation" });
    } finally {
      recoveryInProgress = false;
    }
  };

  evaluate((options.getLeases ?? getActiveMicLeases)());
  return (options.subscribe ?? subscribeMicLeases)(evaluate);
}
