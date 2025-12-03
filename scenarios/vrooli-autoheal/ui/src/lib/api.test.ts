// Tests for API helper functions - especially status classification decision helpers
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002]
import { describe, it, expect } from "vitest";
import {
  groupChecksByStatus,
  sortChecksBySeverity,
  overallStatusFromSummary,
  statusToEmoji,
  STATUS_SEVERITY,
  type HealthResult,
  type HealthSummary,
} from "./api";

describe("Status Classification Helpers", () => {
  // Test fixtures
  const mockChecks: HealthResult[] = [
    {
      checkId: "check-ok-1",
      status: "ok",
      message: "All good",
      timestamp: "2025-01-01T00:00:00Z",
      duration: 10,
    },
    {
      checkId: "check-warning-1",
      status: "warning",
      message: "Something might be wrong",
      timestamp: "2025-01-01T00:00:00Z",
      duration: 15,
    },
    {
      checkId: "check-critical-1",
      status: "critical",
      message: "System is down",
      timestamp: "2025-01-01T00:00:00Z",
      duration: 5,
    },
    {
      checkId: "check-ok-2",
      status: "ok",
      message: "Also good",
      timestamp: "2025-01-01T00:00:00Z",
      duration: 12,
    },
    {
      checkId: "check-warning-2",
      status: "warning",
      message: "Another warning",
      timestamp: "2025-01-01T00:00:00Z",
      duration: 8,
    },
  ];

  describe("groupChecksByStatus", () => {
    it("groups checks correctly by status", () => {
      const grouped = groupChecksByStatus(mockChecks);

      expect(grouped.ok).toHaveLength(2);
      expect(grouped.warning).toHaveLength(2);
      expect(grouped.critical).toHaveLength(1);
    });

    it("returns correct check IDs in each group", () => {
      const grouped = groupChecksByStatus(mockChecks);

      expect(grouped.ok.map((c) => c.checkId)).toEqual([
        "check-ok-1",
        "check-ok-2",
      ]);
      expect(grouped.warning.map((c) => c.checkId)).toEqual([
        "check-warning-1",
        "check-warning-2",
      ]);
      expect(grouped.critical.map((c) => c.checkId)).toEqual([
        "check-critical-1",
      ]);
    });

    it("handles empty array", () => {
      const grouped = groupChecksByStatus([]);

      expect(grouped.ok).toHaveLength(0);
      expect(grouped.warning).toHaveLength(0);
      expect(grouped.critical).toHaveLength(0);
    });

    it("handles all same status", () => {
      const allOk: HealthResult[] = [
        { checkId: "a", status: "ok", message: "", timestamp: "", duration: 0 },
        { checkId: "b", status: "ok", message: "", timestamp: "", duration: 0 },
      ];
      const grouped = groupChecksByStatus(allOk);

      expect(grouped.ok).toHaveLength(2);
      expect(grouped.warning).toHaveLength(0);
      expect(grouped.critical).toHaveLength(0);
    });
  });

  describe("sortChecksBySeverity", () => {
    it("sorts critical first, then warning, then ok", () => {
      const sorted = sortChecksBySeverity(mockChecks);

      expect(sorted[0].status).toBe("critical");
      expect(sorted[1].status).toBe("warning");
      expect(sorted[2].status).toBe("warning");
      expect(sorted[3].status).toBe("ok");
      expect(sorted[4].status).toBe("ok");
    });

    it("does not mutate original array", () => {
      const original = [...mockChecks];
      sortChecksBySeverity(mockChecks);

      expect(mockChecks).toEqual(original);
    });

    it("handles empty array", () => {
      const sorted = sortChecksBySeverity([]);
      expect(sorted).toHaveLength(0);
    });
  });

  describe("overallStatusFromSummary", () => {
    it("returns critical when any critical present", () => {
      const summary: HealthSummary = {
        total: 5,
        ok: 3,
        warning: 1,
        critical: 1,
      };
      expect(overallStatusFromSummary(summary)).toBe("critical");
    });

    it("returns warning when warnings but no critical", () => {
      const summary: HealthSummary = {
        total: 5,
        ok: 3,
        warning: 2,
        critical: 0,
      };
      expect(overallStatusFromSummary(summary)).toBe("warning");
    });

    it("returns ok when all checks are ok", () => {
      const summary: HealthSummary = {
        total: 5,
        ok: 5,
        warning: 0,
        critical: 0,
      };
      expect(overallStatusFromSummary(summary)).toBe("ok");
    });

    it("returns ok for empty summary", () => {
      const summary: HealthSummary = {
        total: 0,
        ok: 0,
        warning: 0,
        critical: 0,
      };
      expect(overallStatusFromSummary(summary)).toBe("ok");
    });
  });

  describe("statusToEmoji", () => {
    it("returns checkmark for ok", () => {
      expect(statusToEmoji("ok")).toBe("\u2713");
    });

    it("returns warning sign for warning", () => {
      expect(statusToEmoji("warning")).toBe("\u26A0");
    });

    it("returns X for critical", () => {
      expect(statusToEmoji("critical")).toBe("\u2717");
    });

    it("returns question mark for unknown status", () => {
      // Force an unknown status value
      expect(statusToEmoji("unknown" as never)).toBe("\u2753");
    });
  });

  describe("STATUS_SEVERITY constant", () => {
    it("has correct severity ordering", () => {
      expect(STATUS_SEVERITY.critical).toBeGreaterThan(STATUS_SEVERITY.warning);
      expect(STATUS_SEVERITY.warning).toBeGreaterThan(STATUS_SEVERITY.ok);
    });

    it("ok has lowest severity", () => {
      expect(STATUS_SEVERITY.ok).toBe(0);
    });

    it("critical has highest severity", () => {
      expect(STATUS_SEVERITY.critical).toBe(2);
    });
  });
});
