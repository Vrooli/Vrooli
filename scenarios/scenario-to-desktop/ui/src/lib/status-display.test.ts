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
  type StageStatus,
  type PipelineStatus,
} from "./status-display";

// ============================================================================
// Stage Status Config
// ============================================================================

describe("STAGE_STATUS_CONFIG", () => {
  it("has config for all stage statuses", () => {
    const statuses: StageStatus[] = ["pending", "running", "completed", "failed", "skipped"];
    for (const status of statuses) {
      expect(STAGE_STATUS_CONFIG[status]).toBeDefined();
      expect(STAGE_STATUS_CONFIG[status].label).toBeTypeOf("string");
      expect(STAGE_STATUS_CONFIG[status].icon).toBeDefined();
      expect(STAGE_STATUS_CONFIG[status].className).toBeTypeOf("string");
    }
  });

  it("has correct labels", () => {
    expect(STAGE_STATUS_CONFIG.pending.label).toBe("Pending");
    expect(STAGE_STATUS_CONFIG.running.label).toBe("Running");
    expect(STAGE_STATUS_CONFIG.completed.label).toBe("Completed");
    expect(STAGE_STATUS_CONFIG.failed.label).toBe("Failed");
    expect(STAGE_STATUS_CONFIG.skipped.label).toBe("Skipped");
  });

  it("has correct icons", () => {
    expect(STAGE_STATUS_CONFIG.pending.icon).toBe(Clock);
    expect(STAGE_STATUS_CONFIG.running.icon).toBe(Clock);
    expect(STAGE_STATUS_CONFIG.completed.icon).toBe(CheckCircle2);
    expect(STAGE_STATUS_CONFIG.failed.icon).toBe(XCircle);
    expect(STAGE_STATUS_CONFIG.skipped.icon).toBe(Circle);
  });
});

describe("HEADER_STATUS_CONFIG", () => {
  it("has config for all stage statuses", () => {
    const statuses: StageStatus[] = ["pending", "running", "completed", "failed", "skipped"];
    for (const status of statuses) {
      expect(HEADER_STATUS_CONFIG[status]).toBeDefined();
      expect(HEADER_STATUS_CONFIG[status].label).toBeTypeOf("string");
      expect(HEADER_STATUS_CONFIG[status].icon).toBeDefined();
      expect(HEADER_STATUS_CONFIG[status].badgeClass).toBeTypeOf("string");
      expect(HEADER_STATUS_CONFIG[status].iconClass).toBeTypeOf("string");
    }
  });

  it("running status has spinner icon", () => {
    expect(HEADER_STATUS_CONFIG.running.icon).toBe(Loader2);
  });
});

// ============================================================================
// getStageStatusDisplay
// ============================================================================

describe("getStageStatusDisplay", () => {
  it("returns correct config for known statuses", () => {
    expect(getStageStatusDisplay("pending")).toEqual(STAGE_STATUS_CONFIG.pending);
    expect(getStageStatusDisplay("running")).toEqual(STAGE_STATUS_CONFIG.running);
    expect(getStageStatusDisplay("completed")).toEqual(STAGE_STATUS_CONFIG.completed);
    expect(getStageStatusDisplay("failed")).toEqual(STAGE_STATUS_CONFIG.failed);
    expect(getStageStatusDisplay("skipped")).toEqual(STAGE_STATUS_CONFIG.skipped);
  });

  it("falls back to pending for unknown statuses", () => {
    expect(getStageStatusDisplay("unknown")).toEqual(STAGE_STATUS_CONFIG.pending);
    expect(getStageStatusDisplay("")).toEqual(STAGE_STATUS_CONFIG.pending);
  });

  it("applies custom labels when provided", () => {
    const result = getStageStatusDisplay("running", { running: "Building" });
    expect(result.label).toBe("Building");
    expect(result.icon).toBe(STAGE_STATUS_CONFIG.running.icon);
    expect(result.className).toBe(STAGE_STATUS_CONFIG.running.className);
  });

  it("does not apply custom label for other statuses", () => {
    const result = getStageStatusDisplay("pending", { running: "Building" });
    expect(result.label).toBe("Pending");
  });

  it("handles multiple custom labels", () => {
    const labels = {
      pending: "Waiting",
      running: "Processing",
      completed: "Done",
    };
    expect(getStageStatusDisplay("pending", labels).label).toBe("Waiting");
    expect(getStageStatusDisplay("running", labels).label).toBe("Processing");
    expect(getStageStatusDisplay("completed", labels).label).toBe("Done");
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
    const result = getPipelineStatusDisplay("pending");
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
    const result = getPipelineStatusDisplay("skipped");
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

  it("returns empty string for empty string", () => {
    expect(formatStageName("")).toBe("");
  });

  it("capitalizes first letter", () => {
    expect(formatStageName("bundle")).toBe("Bundle");
    expect(formatStageName("generate")).toBe("Generate");
    expect(formatStageName("build")).toBe("Build");
  });

  it("preserves rest of the string", () => {
    expect(formatStageName("smokeTest")).toBe("SmokeTest");
    expect(formatStageName("UPPER")).toBe("UPPER");
  });

  it("handles single character", () => {
    expect(formatStageName("a")).toBe("A");
    expect(formatStageName("B")).toBe("B");
  });
});
