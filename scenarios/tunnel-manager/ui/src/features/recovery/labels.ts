import {
  RecoveryStatus,
  EventOutcome,
} from "@vrooli/proto-types/tunnel-manager/v1/recovery/recovery_pb";

import { strings } from "../../consts/strings";
import type { BadgeTone } from "../../components/ui/StatusBadge";

type StatusKey = (typeof strings.recovery.status)[keyof typeof strings.recovery.status];
type OutcomeKey = (typeof strings.recovery.outcome)[keyof typeof strings.recovery.outcome];

/** Map a recovery state-machine phase to its translation key. */
export function recoveryStatusLabel(status: RecoveryStatus): StatusKey {
  switch (status) {
    case RecoveryStatus.IDLE:
      return strings.recovery.status.idle;
    case RecoveryStatus.MONITORING:
      return strings.recovery.status.monitoring;
    case RecoveryStatus.RECOVERING:
      return strings.recovery.status.recovering;
    case RecoveryStatus.CIRCUIT_OPEN:
      return strings.recovery.status.circuitOpen;
    default:
      return strings.recovery.status.unknown;
  }
}

/** Map a recovery phase to a badge tone. */
export function recoveryStatusTone(status: RecoveryStatus): BadgeTone {
  switch (status) {
    case RecoveryStatus.IDLE:
    case RecoveryStatus.MONITORING:
      return "success";
    case RecoveryStatus.RECOVERING:
      return "warning";
    case RecoveryStatus.CIRCUIT_OPEN:
      return "danger";
    default:
      return "neutral";
  }
}

/** Map a recovery-event outcome to its translation key. */
export function eventOutcomeLabel(outcome: EventOutcome): OutcomeKey {
  switch (outcome) {
    case EventOutcome.SUCCESS:
      return strings.recovery.outcome.success;
    case EventOutcome.FAILURE:
      return strings.recovery.outcome.failure;
    case EventOutcome.SKIPPED:
      return strings.recovery.outcome.skipped;
    default:
      return strings.recovery.outcome.unknown;
  }
}

/** Map a recovery-event outcome to a badge tone. */
export function eventOutcomeTone(outcome: EventOutcome): BadgeTone {
  switch (outcome) {
    case EventOutcome.SUCCESS:
      return "success";
    case EventOutcome.FAILURE:
      return "danger";
    case EventOutcome.SKIPPED:
      return "neutral";
    default:
      return "neutral";
  }
}
