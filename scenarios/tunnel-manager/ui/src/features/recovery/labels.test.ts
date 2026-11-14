/**
 * Exhaustive coverage of the recovery enum → label / tone maps, including the
 * UNSPECIFIED / default arms.
 */
import { describe, expect, it } from "vitest";
import {
  RecoveryStatus,
  EventOutcome,
} from "@vrooli/proto-types/tunnel-manager/v1/recovery/recovery_pb";

import { strings } from "../../consts/strings";
import {
  recoveryStatusLabel,
  recoveryStatusTone,
  eventOutcomeLabel,
  eventOutcomeTone,
} from "./labels";

describe("recovery labels", () => {
  it("maps every status to its label", () => {
    expect(recoveryStatusLabel(RecoveryStatus.IDLE)).toBe(strings.recovery.status.idle);
    expect(recoveryStatusLabel(RecoveryStatus.MONITORING)).toBe(strings.recovery.status.monitoring);
    expect(recoveryStatusLabel(RecoveryStatus.RECOVERING)).toBe(strings.recovery.status.recovering);
    expect(recoveryStatusLabel(RecoveryStatus.CIRCUIT_OPEN)).toBe(strings.recovery.status.circuitOpen);
    expect(recoveryStatusLabel(RecoveryStatus.UNSPECIFIED)).toBe(strings.recovery.status.unknown);
  });

  it("maps every status to its tone", () => {
    expect(recoveryStatusTone(RecoveryStatus.IDLE)).toBe("success");
    expect(recoveryStatusTone(RecoveryStatus.MONITORING)).toBe("success");
    expect(recoveryStatusTone(RecoveryStatus.RECOVERING)).toBe("warning");
    expect(recoveryStatusTone(RecoveryStatus.CIRCUIT_OPEN)).toBe("danger");
    expect(recoveryStatusTone(RecoveryStatus.UNSPECIFIED)).toBe("neutral");
  });

  it("maps every outcome to its label", () => {
    expect(eventOutcomeLabel(EventOutcome.SUCCESS)).toBe(strings.recovery.outcome.success);
    expect(eventOutcomeLabel(EventOutcome.FAILURE)).toBe(strings.recovery.outcome.failure);
    expect(eventOutcomeLabel(EventOutcome.SKIPPED)).toBe(strings.recovery.outcome.skipped);
    expect(eventOutcomeLabel(EventOutcome.UNSPECIFIED)).toBe(strings.recovery.outcome.unknown);
  });

  it("maps every outcome to its tone", () => {
    expect(eventOutcomeTone(EventOutcome.SUCCESS)).toBe("success");
    expect(eventOutcomeTone(EventOutcome.FAILURE)).toBe("danger");
    expect(eventOutcomeTone(EventOutcome.SKIPPED)).toBe("neutral");
    expect(eventOutcomeTone(EventOutcome.UNSPECIFIED)).toBe("neutral");
  });
});
