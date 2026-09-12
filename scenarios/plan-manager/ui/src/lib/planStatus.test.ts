/**
 * planStatus tests — descriptor totality, phase counting, and bigint clamping.
 */
import { describe, expect, it } from "vitest";

import {
  bigintToNumber,
  countPhases,
  phaseStatusDescriptor,
  planStatusDescriptor,
  stalenessDescriptor,
  verdictDescriptor,
} from "./planStatus";
import {
  PhaseStatus,
  PlanStatus,
  StalenessTier,
  ValidationVerdict,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

describe("planStatus descriptors", () => {
  it("returns a descriptor for every plan status", () => {
    const statuses = Object.values(PlanStatus).filter(
      (v): v is PlanStatus => typeof v === "number",
    );
    for (const status of statuses) {
      expect(planStatusDescriptor(status).labelKey).toBeTruthy();
    }
  });

  it("returns a descriptor for every phase status", () => {
    const statuses = Object.values(PhaseStatus).filter(
      (v): v is PhaseStatus => typeof v === "number",
    );
    for (const status of statuses) {
      expect(phaseStatusDescriptor(status).labelKey).toBeTruthy();
    }
  });

  it("returns a descriptor for every staleness tier and verdict", () => {
    expect(stalenessDescriptor(StalenessTier.FRESH).tone).toBe("success");
    expect(stalenessDescriptor(StalenessTier.DEFINITELY_STALE).tone).toBe("danger");
    expect(verdictDescriptor(ValidationVerdict.PASS).tone).toBe("success");
    expect(verdictDescriptor(ValidationVerdict.UNKNOWN).tone).toBe("warning");
  });
});

describe("countPhases", () => {
  it("counts each status bucket", () => {
    const counts = countPhases([
      PhaseStatus.DONE,
      PhaseStatus.DONE,
      PhaseStatus.ACTIVE,
      PhaseStatus.TODO,
      PhaseStatus.BLOCKED,
    ]);
    expect(counts).toEqual({ total: 5, todo: 1, active: 1, done: 2, blocked: 1 });
  });
});

describe("bigintToNumber", () => {
  it("converts in-range values exactly", () => {
    expect(bigintToNumber(45000n)).toBe(45000);
    expect(bigintToNumber(0n)).toBe(0);
  });

  it("clamps values beyond the safe-integer range", () => {
    expect(bigintToNumber(BigInt(Number.MAX_SAFE_INTEGER) + 10n)).toBe(Number.MAX_SAFE_INTEGER);
    expect(bigintToNumber(BigInt(Number.MIN_SAFE_INTEGER) - 10n)).toBe(Number.MIN_SAFE_INTEGER);
  });
});
