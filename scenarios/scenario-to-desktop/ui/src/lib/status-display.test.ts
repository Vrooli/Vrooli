/**
 * Tests for status display utility functions.
 */

import { describe, it, expect } from "vitest";
import {
  Loader2,
  CheckCircle2,
  XCircle,
  Circle,
  Clock,
  AlertCircle,
  Play,
} from "lucide-react";
import {
  STAGE_STATUS_CONFIG,
  HEADER_STATUS_CONFIG,
  getStageStatusDisplay,
  getPipelineStatusDisplay,
  formatStageName,
} from "./status-display";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

// ============================================================================
// Stage Status Config
// ============================================================================

describe("STAGE_STATUS_CONFIG", () => {
  it("has config for all stage statuses", () => {
    const statuses: StageStatus[] = [
      StageStatus.PENDING,
      StageStatus.RUNNING,
      StageStatus.COMPLETED,
      StageStatus.FAILED,
      StageStatus.SKIPPED,
    ];
    for (const status of statuses) {
      expect(STAGE_STATUS_CONFIG[status]).toBeDefined();
      expect(STAGE_STATUS_CONFIG[status]?.label).toBeTypeOf("string");
      expect(STAGE_STATUS_CONFIG[status]?.icon).toBeDefined();
      expect(STAGE_STATUS_CONFIG[status]?.className).toBeTypeOf("string");
    }
  });

  it("has correct labels", () => {
    expect(STAGE_STATUS_CONFIG[StageStatus.PENDING]?.label).toBe("Pending");
    expect(STAGE_STATUS_CONFIG[StageStatus.RUNNING]?.label).toBe("Running");
    expect(STAGE_STATUS_CONFIG[StageStatus.COMPLETED]?.label).toBe("Completed");
    expect(STAGE_STATUS_CONFIG[StageStatus.FAILED]?.label).toBe("Failed");
    expect(STAGE_STATUS_CONFIG[StageStatus.SKIPPED]?.label).toBe("Skipped");
  });

  it("has correct icons", () => {
    expect(STAGE_STATUS_CONFIG[StageStatus.PENDING]?.icon).toBe(Clock);
    expect(STAGE_STATUS_CONFIG[StageStatus.RUNNING]?.icon).toBe(Clock);
    expect(STAGE_STATUS_CONFIG[StageStatus.COMPLETED]?.icon).toBe(CheckCircle2);
    expect(STAGE_STATUS_CONFIG[StageStatus.FAILED]?.icon).toBe(XCircle);
    expect(STAGE_STATUS_CONFIG[StageStatus.SKIPPED]?.icon).toBe(Circle);
  });
});

describe("HEADER_STATUS_CONFIG", () => {
  it("has config for all stage statuses", () => {
    const statuses: StageStatus[] = [
      StageStatus.PENDING,
      StageStatus.RUNNING,
      StageStatus.COMPLETED,
      StageStatus.FAILED,
      StageStatus.SKIPPED,
    ];
    for (const status of statuses) {
      expect(HEADER_STATUS_CONFIG[status]).toBeDefined();
      expect(HEADER_STATUS_CONFIG[status]?.label).toBeTypeOf("string");
      expect(HEADER_STATUS_CONFIG[status]?.icon).toBeDefined();
      expect(HEADER_STATUS_CONFIG[status]?.badgeClass).toBeTypeOf("string");
      expect(HEADER_STATUS_CONFIG[status]?.iconClass).toBeTypeOf("string");
    }
  });

  it("running status has spinner icon", () => {
    expect(HEADER_STATUS_CONFIG[StageStatus.RUNNING]?.icon).toBe(Loader2);
  });
});

// ============================================================================
// getStageStatusDisplay
// ============================================================================

describe("getStageStatusDisplay", () => {
  it("returns correct config for known statuses", () => {
    expect(getStageStatusDisplay(StageStatus.PENDING)).toEqual(
      STAGE_STATUS_CONFIG[StageStatus.PENDING],
    );
    expect(getStageStatusDisplay(StageStatus.RUNNING)).toEqual(
      STAGE_STATUS_CONFIG[StageStatus.RUNNING],
    );
    expect(getStageStatusDisplay(StageStatus.COMPLETED)).toEqual(
      STAGE_STATUS_CONFIG[StageStatus.COMPLETED],
    );
    expect(getStageStatusDisplay(StageStatus.FAILED)).toEqual(
      STAGE_STATUS_CONFIG[StageStatus.FAILED],
    );
    expect(getStageStatusDisplay(StageStatus.SKIPPED)).toEqual(
      STAGE_STATUS_CONFIG[StageStatus.SKIPPED],
    );
  });

  it("falls back to pending for an unspecified enum", () => {
    expect(getStageStatusDisplay(StageStatus.UNSPECIFIED)).toEqual(
      STAGE_STATUS_CONFIG[StageStatus.PENDING],
    );
  });

  it("applies custom labels when provided", () => {
    const result = getStageStatusDisplay(StageStatus.RUNNING, {
      [StageStatus.RUNNING]: "Building",
    });
    expect(result.label).toBe("Building");
    expect(result.icon).toBe(STAGE_STATUS_CONFIG[StageStatus.RUNNING]?.icon);
    expect(result.className).toBe(
      STAGE_STATUS_CONFIG[StageStatus.RUNNING]?.className,
    );
  });

  it("does not apply custom label for other statuses", () => {
    const result = getStageStatusDisplay(StageStatus.PENDING, {
      [StageStatus.RUNNING]: "Building",
    });
    expect(result.label).toBe("Pending");
  });

  it("handles multiple custom labels", () => {
    const labels = {
      [StageStatus.PENDING]: "Waiting",
      [StageStatus.RUNNING]: "Processing",
      [StageStatus.COMPLETED]: "Done",
    };
    expect(getStageStatusDisplay(StageStatus.PENDING, labels).label).toBe(
      "Waiting",
    );
    expect(getStageStatusDisplay(StageStatus.RUNNING, labels).label).toBe(
      "Processing",
    );
    expect(getStageStatusDisplay(StageStatus.COMPLETED, labels).label).toBe(
      "Done",
    );
  });
});

// ============================================================================
// getPipelineStatusDisplay
// ============================================================================

describe("getPipelineStatusDisplay", () => {
  it("returns running config for running status", () => {
    const result = getPipelineStatusDisplay("running");
    expect(result.label).toBe("Running");
    expect(result.icon).toBe(Loader2);
    expect(result.className).toContain("blue");
  });

  it("keeps enum-backed pipeline statuses and an idle pipeline distinct", () => {
    expect(getPipelineStatusDisplay("idle").label).toBe("Ready");
    expect(getPipelineStatusDisplay(StageStatus.RUNNING).label).toBe("Running");
    expect(getPipelineStatusDisplay(StageStatus.COMPLETED).label).toBe(
      "Completed",
    );
    expect(getPipelineStatusDisplay(StageStatus.FAILED).label).toBe("Failed");
  });

  it("returns running config for starting status", () => {
    const result = getPipelineStatusDisplay("starting");
    expect(result.label).toBe("Running");
    expect(result.icon).toBe(Loader2);
  });

  it("returns completed config for completed status", () => {
    const result = getPipelineStatusDisplay("completed");
    expect(result.label).toBe("Completed");
    expect(result.icon).toBe(CheckCircle2);
    expect(result.className).toContain("green");
  });

  it("returns failed config for failed status", () => {
    const result = getPipelineStatusDisplay("failed");
    expect(result.label).toBe("Failed");
    expect(result.icon).toBe(AlertCircle);
    expect(result.className).toContain("red");
  });

  it("returns cancelled config for cancelled status", () => {
    const result = getPipelineStatusDisplay("cancelled");
    expect(result.label).toBe("Cancelled");
    expect(result.icon).toBe(AlertCircle);
    expect(result.className).toContain("yellow");
  });

  it("returns pending config for pending status", () => {
    const result = getPipelineStatusDisplay(StageStatus.PENDING);
    expect(result.label).toBe("Pending");
    expect(result.icon).toBe(Circle);
    expect(result.className).toContain("slate");
  });

  it("returns no pipeline config for null", () => {
    const result = getPipelineStatusDisplay(null);
    expect(result.label).toBe("No Pipeline");
    expect(result.icon).toBe(Play);
  });

  it("returns no pipeline config for undefined", () => {
    const result = getPipelineStatusDisplay(undefined);
    expect(result.label).toBe("No Pipeline");
    expect(result.icon).toBe(Play);
  });

  it("returns no pipeline config for skipped status", () => {
    const result = getPipelineStatusDisplay(StageStatus.SKIPPED);
    expect(result.label).toBe("No Pipeline");
  });
});

// ============================================================================
// formatStageName
// ============================================================================

describe("formatStageName", () => {
  it("returns empty string for null", () => {
    expect(formatStageName(null)).toBe("");
  });

  it("returns empty string for an unspecified enum", () => {
    expect(formatStageName(StageName.UNSPECIFIED)).toBe("");
  });

  it("capitalizes first letter", () => {
    expect(formatStageName(StageName.BUNDLE)).toBe("Bundle");
    expect(formatStageName(StageName.GENERATE)).toBe("Generate");
    expect(formatStageName(StageName.BUILD)).toBe("Build");
  });

  it("formats the compound smoke-test stage", () => {
    expect(formatStageName(StageName.SMOKE_TEST)).toBe("Smoke test");
  });

  it("formats every remaining pipeline stage name", () => {
    expect(formatStageName(StageName.RESOLVE_DEPLOYMENT)).toBe(
      "Resolve deployment",
    );
    expect(formatStageName(StageName.PREFLIGHT)).toBe("Preflight");
    expect(formatStageName(StageName.DEPLOY)).toBe("Deploy");
  });
});
