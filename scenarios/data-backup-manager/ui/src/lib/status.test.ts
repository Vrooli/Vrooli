import { describe, expect, it } from "vitest";

import {
  RunStatus,
  TargetOutcomeStatus,
} from "@vrooli/proto-types/data-backup-manager/v1/runs/runs_pb";
import { RestoreStatus } from "@vrooli/proto-types/data-backup-manager/v1/restores/restores_pb";
import { UsageState } from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";

import {
  isRunInFlight,
  outcomeMeta,
  restoreStatusMeta,
  runStatusMeta,
  usageMeta,
  verifiedMeta,
} from "./status";

describe("runStatusMeta", () => {
  it("maps completed to success and failed to danger", () => {
    expect(runStatusMeta(RunStatus.COMPLETED)).toEqual({ tone: "success", slug: "completed" });
    expect(runStatusMeta(RunStatus.FAILED)).toEqual({ tone: "danger", slug: "failed" });
  });

  it("maps partial failure to warning, not danger (some targets succeeded)", () => {
    expect(runStatusMeta(RunStatus.PARTIAL_FAILED)).toEqual({
      tone: "warning",
      slug: "partialFailed",
    });
  });
});

describe("isRunInFlight", () => {
  it("is true for pending/capturing/snapshotting and false for terminal", () => {
    expect(isRunInFlight(RunStatus.PENDING)).toBe(true);
    expect(isRunInFlight(RunStatus.CAPTURING)).toBe(true);
    expect(isRunInFlight(RunStatus.SNAPSHOTTING)).toBe(true);
    expect(isRunInFlight(RunStatus.COMPLETED)).toBe(false);
    expect(isRunInFlight(RunStatus.PARTIAL_FAILED)).toBe(false);
  });
});

describe("outcomeMeta", () => {
  it("keeps cap-blocked distinct from failed (warning, not danger)", () => {
    expect(outcomeMeta(TargetOutcomeStatus.BLOCKED)).toEqual({ tone: "warning", slug: "blocked" });
    expect(outcomeMeta(TargetOutcomeStatus.FAILED)).toEqual({ tone: "danger", slug: "failed" });
    expect(outcomeMeta(TargetOutcomeStatus.SUCCEEDED)).toEqual({
      tone: "success",
      slug: "succeeded",
    });
  });
});

describe("restoreStatusMeta", () => {
  it("maps verified and restored to success, failed to danger", () => {
    expect(restoreStatusMeta(RestoreStatus.VERIFIED).tone).toBe("success");
    expect(restoreStatusMeta(RestoreStatus.RESTORED).tone).toBe("success");
    expect(restoreStatusMeta(RestoreStatus.FAILED).tone).toBe("danger");
  });
});

describe("usageMeta", () => {
  it("maps within/near/over to success/warning/danger", () => {
    expect(usageMeta(UsageState.WITHIN).tone).toBe("success");
    expect(usageMeta(UsageState.NEAR).tone).toBe("warning");
    expect(usageMeta(UsageState.OVER).tone).toBe("danger");
  });
});

describe("verifiedMeta — the product's spine", () => {
  const now = new Date("2026-05-01T00:00:00Z");
  const day = 86_400_000;

  it("flags a backed-up-but-never-verified target as unverified (warning)", () => {
    expect(verifiedMeta(undefined, now)).toEqual({ tone: "warning", slug: "unverified" });
  });

  it("marks a recent verify as verified (success)", () => {
    const recent = new Date(now.getTime() - 2 * day);
    expect(verifiedMeta(recent, now)).toEqual({ tone: "success", slug: "verified" });
  });

  it("re-flags a long-ago verify as stale (warning)", () => {
    const old = new Date(now.getTime() - 40 * day);
    expect(verifiedMeta(old, now)).toEqual({ tone: "warning", slug: "stale" });
  });
});
