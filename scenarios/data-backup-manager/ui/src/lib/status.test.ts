import { describe, expect, it } from "vitest";

import {
	RunStatus,
	TargetOutcomeStatus,
	TriggerSource,
} from "@vrooli/proto-types/data-backup-manager/v1/runs/runs_pb";
import { RestoreMode, RestoreStatus } from "@vrooli/proto-types/data-backup-manager/v1/restores/restores_pb";
import { BackendKind, CapPolicy, UsageState } from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";
import { SourceKind } from "@vrooli/proto-types/data-backup-manager/v1/sources/sources_pb";
import { DriveClass } from "@vrooli/proto-types/data-backup-manager/v1/discovery/discovery_pb";

import {
	isRunInFlight,
	isRestoreInFlight,
	restoreModeSlug,
	outcomeMeta,
	restoreStatusMeta,
	runStatusMeta,
	triggerSlug,
	usageMeta,
	capPolicySlug,
	backendSlug,
	sourceKindSlug,
	driveClassMeta,
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

describe("status vocabulary completeness", () => {
	it("maps intermediate, source, destination, and restore states", () => {
		expect(runStatusMeta(RunStatus.PENDING).slug).toBe("pending");
		expect(runStatusMeta(RunStatus.CAPTURING).slug).toBe("capturing");
		expect(runStatusMeta(RunStatus.SNAPSHOTTING).slug).toBe("snapshotting");
		expect(runStatusMeta(RunStatus.UNSPECIFIED).slug).toBe("unknown");
		expect(triggerSlug(TriggerSource.SCHEDULER)).toBe("scheduler");
		expect(triggerSlug(TriggerSource.MANUAL)).toBe("manual");
		expect(isRestoreInFlight(RestoreStatus.REQUESTED)).toBe(true);
		expect(isRestoreInFlight(RestoreStatus.RESTORING)).toBe(true);
		expect(isRestoreInFlight(RestoreStatus.VERIFYING)).toBe(true);
		expect(isRestoreInFlight(RestoreStatus.FAILED)).toBe(false);
		expect(restoreModeSlug(RestoreMode.RESTORE)).toBe("restore");
		expect(restoreModeSlug(RestoreMode.VERIFY)).toBe("verify");
		expect(capPolicySlug(CapPolicy.ALERT_BLOCK)).toBe("alertBlock");
		expect(capPolicySlug(CapPolicy.ALERT_ONLY)).toBe("alertOnly");
		expect(backendSlug(BackendKind.FILESYSTEM)).toBe("filesystem");
		expect(backendSlug(BackendKind.S3)).toBe("s3");
		expect(sourceKindSlug(SourceKind.FILESYSTEM)).toBe("filesystem");
		expect(sourceKindSlug(SourceKind.SQLITE)).toBe("sqlite");
		expect(sourceKindSlug(SourceKind.POSTGRES)).toBe("postgres");
		expect(sourceKindSlug(SourceKind.REDIS)).toBe("redis");
		expect(sourceKindSlug(SourceKind.QDRANT)).toBe("qdrant");
		expect(sourceKindSlug(SourceKind.OBJECT_STORAGE)).toBe("objectStorage");
		expect(driveClassMeta(DriveClass.REMOVABLE).slug).toBe("removable");
		expect(driveClassMeta(DriveClass.FIXED).slug).toBe("fixed");
		expect(driveClassMeta(DriveClass.NETWORK).slug).toBe("network");
		expect(outcomeMeta(999 as TargetOutcomeStatus).slug).toBe("unknown");
		expect(triggerSlug(999 as TriggerSource)).toBe("unknown");
		expect(restoreStatusMeta(999 as RestoreStatus).slug).toBe("unknown");
		expect(restoreModeSlug(999 as RestoreMode)).toBe("unknown");
		expect(usageMeta(999 as UsageState).slug).toBe("unknown");
		expect(capPolicySlug(999 as CapPolicy)).toBe("unknown");
		expect(backendSlug(999 as BackendKind)).toBe("unknown");
		expect(sourceKindSlug(999 as SourceKind)).toBe("unknown");
		expect(driveClassMeta(999 as DriveClass).slug).toBe("unknown");
	});
});
