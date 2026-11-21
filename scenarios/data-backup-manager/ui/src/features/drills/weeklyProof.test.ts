import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";

import { DrillStatus } from "../../api/drills";
import { RecoveryDrillSchema } from "@vrooli/proto-types/data-backup-manager/v1/drills/drills_pb";
import { PlanSchema } from "@vrooli/proto-types/data-backup-manager/v1/plans/plans_pb";
import { evaluateWeeklyProof, weeklyProofUnits } from "./weeklyProof";

const now = new Date("2026-11-14T12:00:00Z");
const plan = (overrides: Record<string, unknown> = {}) => create(PlanSchema, {
  id: "plan-1",
  enabled: true,
  targetIds: ["target-1"],
  destinationIds: ["dest-1"],
  recoveryDrillSchedule: "168h",
  ...overrides,
});
const receipt = (id: string, status: DrillStatus, requestedAt: string, overrides: Record<string, unknown> = {}) => create(RecoveryDrillSchema, {
  id,
  planId: "plan-1",
  targetId: "target-1",
  destinationId: "dest-1",
  status,
  scheduled: true,
  requestedAt: timestampFromDate(new Date(requestedAt)),
  ...overrides,
});

describe("evaluateWeeklyProof", () => {
  it("requires a recent scheduled verified receipt completed within seven days", () => {
    const recent = receipt("recent", DrillStatus.VERIFIED, "2026-11-13T10:00:00Z", {
      finishedAt: timestampFromDate(new Date("2026-11-13T10:05:00Z")),
    });
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [recent], now).state).toBe("verified");
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [], now).state).toBe("missing");

    const manual = { ...recent, scheduled: false };
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [manual], now).state).toBe("missing");

    const stale = receipt("stale", DrillStatus.VERIFIED, "2026-11-01T10:00:00Z", {
      finishedAt: timestampFromDate(new Date("2026-11-01T10:05:00Z")),
    });
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [stale], now).state).toBe("stale");
  });

  it("does not let an older success hide a later scheduled failure or pending drill", () => {
    const verified = receipt("verified", DrillStatus.VERIFIED, "2026-11-13T10:00:00Z", {
      finishedAt: timestampFromDate(new Date("2026-11-13T10:05:00Z")),
    });
    const failed = receipt("failed", DrillStatus.FAILED, "2026-11-14T11:00:00Z");
    const pending = receipt("pending", DrillStatus.RUNNING, "2026-11-14T11:30:00Z");
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [verified, failed], now).state).toBe("failed");
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [verified, pending], now).state).toBe("pending");
  });

  it("excludes retired receipts, manual-only plans and disabled plans", () => {
    const retired = receipt("retired", DrillStatus.VERIFIED, "2026-11-13T10:00:00Z", {
      finishedAt: timestampFromDate(new Date("2026-11-13T10:05:00Z")),
      retiredAt: timestampFromDate(new Date("2026-11-14T11:00:00Z")),
    });
    expect(evaluateWeeklyProof(plan(), "target-1", "dest-1", [retired], now).state).toBe("retired");
    expect(evaluateWeeklyProof(plan({ recoveryDrillSchedule: "manual" }), "target-1", "dest-1", [], now).state).toBe("manual");
    expect(evaluateWeeklyProof(plan({ enabled: false }), "target-1", "dest-1", [], now).state).toBe("disabled");
  });

  it("counts the configured target × destination matrix while excluding manual and disabled plans", () => {
    const matrix = plan({ targetIds: ["target-1", "target-2"], destinationIds: ["dest-1", "dest-2"] });
    const manual = plan({ id: "manual", recoveryDrillSchedule: "manual" });
    const disabled = plan({ id: "disabled", enabled: false });
    const verified = receipt("verified", DrillStatus.VERIFIED, "2026-11-13T10:00:00Z", {
      finishedAt: timestampFromDate(new Date("2026-11-13T10:05:00Z")),
    });
    const units = weeklyProofUnits([matrix, manual, disabled], [verified], now);
    const eligible = units.filter((unit) => unit.state !== "manual" && unit.state !== "disabled");
    expect(eligible).toHaveLength(4);
    expect(eligible.filter((unit) => unit.state === "verified")).toHaveLength(1);
    expect(units.filter((unit) => unit.state === "manual")).toHaveLength(1);
    expect(units.filter((unit) => unit.state === "disabled")).toHaveLength(1);
  });
});
