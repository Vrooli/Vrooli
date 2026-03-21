import { describe, it, expect } from "vitest";
import {
  BACKLOG_STATUS_COLORS,
  EXECUTION_STATUS_COLORS,
  formatExecutionMode,
  formatExecutionStatus,
  SCENARIO_STATUS_ICONS,
  SCENARIO_STATUS_COLORS,
  formatBacklogStatus,
} from "./constants";
import type { BacklogStatus, ExecutionMode, ExecutionStatus, ScenarioStatus } from "./domain";

/**
 * Constants module tests.
 *
 * These tests verify that the status-to-display mappings are complete
 * and cover all possible domain status values. This prevents runtime
 * errors when encountering an unmapped status.
 *
 * DOC: docs/internal/SEAMS.md#decision-points
 * [REQ:REQ-P0-008] Status display mappings for UI
 */
describe("Constants - Decision Boundaries", () => {
  // All possible backlog statuses from the domain type
  const ALL_BACKLOG_STATUSES: BacklogStatus[] = [
    "backlog",
    "researching",
    "ready",
    "queued",
    "in_progress",
    "completed",
    "archived",
  ];

  // All possible scenario statuses from the domain type
  const ALL_SCENARIO_STATUSES: ScenarioStatus[] = [
    "running",
    "stopped",
    "error",
    "unknown",
  ];

  const ALL_EXECUTION_STATUSES: ExecutionStatus[] = [
    "pending",
    "scheduled",
    "running",
    "completed",
    "failed",
    "canceled",
  ];

  const ALL_EXECUTION_MODES: ExecutionMode[] = ["manual", "scheduled", "yolo"];

  describe("BACKLOG_STATUS_COLORS", () => {
    it("has a color mapping for every backlog status", () => {
      ALL_BACKLOG_STATUSES.forEach((status) => {
        expect(BACKLOG_STATUS_COLORS[status]).toBeDefined();
        expect(typeof BACKLOG_STATUS_COLORS[status]).toBe("string");
      });
    });

    it("returns valid Tailwind background classes", () => {
      Object.values(BACKLOG_STATUS_COLORS).forEach((color) => {
        expect(color).toMatch(/^bg-/);
      });
    });

    it("uses distinct colors for different status meanings", () => {
      // Active/positive statuses should use distinct colors from inactive
      const activeColors = [
        BACKLOG_STATUS_COLORS.researching,
        BACKLOG_STATUS_COLORS.in_progress,
      ];
      const completedColor = BACKLOG_STATUS_COLORS.completed;
      const archivedColor = BACKLOG_STATUS_COLORS.archived;

      // Completed should be visually different from archived
      expect(completedColor).not.toBe(archivedColor);

      // Active statuses should be distinct
      expect(new Set(activeColors).size).toBe(activeColors.length);
    });
  });

  describe("SCENARIO_STATUS_ICONS", () => {
    it("has an icon mapping for every scenario status", () => {
      ALL_SCENARIO_STATUSES.forEach((status) => {
        expect(SCENARIO_STATUS_ICONS[status]).toBeDefined();
        // Icons are React forward-ref components (objects with $$typeof)
        expect(SCENARIO_STATUS_ICONS[status]).toHaveProperty("$$typeof");
      });
    });

    it("uses a check-related icon for running status (positive indicator)", () => {
      // Lucide icons may have different display names (CircleCheckBig, CheckCircle, etc.)
      const displayName = SCENARIO_STATUS_ICONS.running.displayName;
      expect(displayName).toMatch(/check/i);
    });

    it("uses an alert-related icon for error status (warning indicator)", () => {
      // Lucide icons may have different display names (CircleAlert, AlertCircle, etc.)
      const displayName = SCENARIO_STATUS_ICONS.error.displayName;
      expect(displayName).toMatch(/alert/i);
    });
  });

  describe("SCENARIO_STATUS_COLORS", () => {
    it("has a color mapping for every scenario status", () => {
      ALL_SCENARIO_STATUSES.forEach((status) => {
        expect(SCENARIO_STATUS_COLORS[status]).toBeDefined();
        expect(typeof SCENARIO_STATUS_COLORS[status]).toBe("string");
      });
    });

    it("returns valid Tailwind text classes", () => {
      Object.values(SCENARIO_STATUS_COLORS).forEach((color) => {
        expect(color).toMatch(/^text-/);
      });
    });

    it("uses semantic colors for status indication", () => {
      // Running = green (positive)
      expect(SCENARIO_STATUS_COLORS.running).toContain("green");
      // Error = red (warning)
      expect(SCENARIO_STATUS_COLORS.error).toContain("red");
      // Stopped/unknown = neutral (gray/slate)
      expect(SCENARIO_STATUS_COLORS.stopped).toMatch(/slate|gray/);
    });
  });

  describe("formatBacklogStatus", () => {
    it("converts underscores to spaces and capitalizes", () => {
      expect(formatBacklogStatus("in_progress")).toBe("In progress");
    });

    it("capitalizes single-word statuses", () => {
      expect(formatBacklogStatus("backlog")).toBe("Backlog");
      expect(formatBacklogStatus("researching")).toBe("Researching");
    });

    it("formats all statuses without errors", () => {
      ALL_BACKLOG_STATUSES.forEach((status) => {
        expect(() => formatBacklogStatus(status)).not.toThrow();
        expect(typeof formatBacklogStatus(status)).toBe("string");
        // All formatted statuses should start with uppercase
        const formatted = formatBacklogStatus(status);
        expect(formatted.charAt(0)).toBe(formatted.charAt(0).toUpperCase());
      });
    });
  });

  describe("EXECUTION_STATUS_COLORS", () => {
    it("has a color mapping for every execution status", () => {
      ALL_EXECUTION_STATUSES.forEach((status) => {
        expect(EXECUTION_STATUS_COLORS[status]).toBeDefined();
        expect(typeof EXECUTION_STATUS_COLORS[status]).toBe("string");
      });
    });

    it("returns valid Tailwind background classes", () => {
      Object.values(EXECUTION_STATUS_COLORS).forEach((color) => {
        expect(color).toMatch(/^bg-/);
      });
    });
  });

  describe("formatExecutionStatus", () => {
    it("formats all execution statuses without errors", () => {
      ALL_EXECUTION_STATUSES.forEach((status) => {
        expect(() => formatExecutionStatus(status)).not.toThrow();
        expect(typeof formatExecutionStatus(status)).toBe("string");
      });
    });
  });

  describe("formatExecutionMode", () => {
    it("formats all execution modes without errors", () => {
      ALL_EXECUTION_MODES.forEach((mode) => {
        expect(() => formatExecutionMode(mode)).not.toThrow();
        expect(typeof formatExecutionMode(mode)).toBe("string");
      });
    });

    it("formats yolo mode as Auto", () => {
      expect(formatExecutionMode("yolo")).toBe("Auto");
    });
  });
});
