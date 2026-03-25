import { describe, expect, it } from "vitest";
import {
  canCancelExecution,
  canFollowUpExecution,
  canRetryExecution,
  canStartExecution,
  EXECUTION_TAB_CONFIG,
  isExecutionActive,
  isExecutionInTab,
  matchesExecutionFilters,
  type ExecutionFilters,
} from "./execution-utils";
import type { ExecutionRecord } from "../types";

const baseRecord: ExecutionRecord = {
  executionId: "exec_123",
  backlogKind: "execute",
  backlogName: "deploy-health-check",
  status: "running",
  mode: "manual",
  operation: "improver",
  startedBy: "swarm-manager-ui",
  runId: "run_456",
  taskId: "task_789",
  createdAt: "2026-02-12T10:00:00.000Z",
  updatedAt: "2026-02-12T10:05:00.000Z",
};

const baseFilters: ExecutionFilters = {
  searchTerm: "",
  statusFilter: "",
  modeFilter: "",
  startedByFilter: "",
  operationFilter: "",
  backlogFilter: "",
  fromFilter: "",
  toFilter: "",
};

describe("execution-utils", () => {
  describe("isExecutionInTab", () => {
    it("matches running items in running tab", () => {
      expect(isExecutionInTab(baseRecord, "running")).toBe(true);
    });

    it("does not match running items in completed tab", () => {
      expect(isExecutionInTab(baseRecord, "completed")).toBe(false);
    });
  });

  describe("matchesExecutionFilters", () => {
    it("matches by search term", () => {
      expect(matchesExecutionFilters(baseRecord, { ...baseFilters, searchTerm: "deploy" })).toBe(true);
    });

    it("filters by status", () => {
      expect(matchesExecutionFilters(baseRecord, { ...baseFilters, statusFilter: "running" })).toBe(true);
      expect(matchesExecutionFilters(baseRecord, { ...baseFilters, statusFilter: "failed" })).toBe(false);
    });

    it("filters by date range", () => {
      expect(matchesExecutionFilters(baseRecord, {
        ...baseFilters,
        fromFilter: "2026-02-12T09:00:00.000Z",
        toFilter: "2026-02-12T11:00:00.000Z",
      })).toBe(true);

      expect(matchesExecutionFilters(baseRecord, {
        ...baseFilters,
        fromFilter: "2026-02-12T10:01:00.000Z",
      })).toBe(false);
    });
  });

  describe("action helpers", () => {
    it("returns true for active executions", () => {
      expect(isExecutionActive(baseRecord)).toBe(true);
    });

    it("returns action availability by status", () => {
      expect(canStartExecution("pending")).toBe(true);
      expect(canStartExecution("running")).toBe(false);
      expect(canCancelExecution("running")).toBe(true);
      expect(canCancelExecution("completed")).toBe(false);
      expect(canRetryExecution("failed")).toBe(true);
      expect(canRetryExecution("canceled")).toBe(false);
    });
  });

  describe("canFollowUpExecution", () => {
    it("returns true for completed, failed, needs_fixup", () => {
      expect(canFollowUpExecution("completed")).toBe(true);
      expect(canFollowUpExecution("failed")).toBe(true);
      expect(canFollowUpExecution("needs_fixup")).toBe(true);
    });

    it("returns false for pending, running, validating, etc.", () => {
      expect(canFollowUpExecution("pending")).toBe(false);
      expect(canFollowUpExecution("running")).toBe(false);
      expect(canFollowUpExecution("validating")).toBe(false);
      expect(canFollowUpExecution("scheduled")).toBe(false);
      expect(canFollowUpExecution("starting")).toBe(false);
      expect(canFollowUpExecution("needs_review")).toBe(false);
      expect(canFollowUpExecution("canceled")).toBe(false);
    });
  });

  describe("isExecutionActive extended", () => {
    it("returns true for validating", () => {
      expect(isExecutionActive({ ...baseRecord, status: "validating" })).toBe(true);
    });

    it("returns false for needs_fixup", () => {
      expect(isExecutionActive({ ...baseRecord, status: "needs_fixup" })).toBe(false);
    });
  });

  describe("EXECUTION_TAB_CONFIG", () => {
    it('"all" tab includes validating and needs_fixup', () => {
      const allTab = EXECUTION_TAB_CONFIG.find((tab) => tab.id === "all");
      expect(allTab).toBeDefined();
      expect(allTab!.statuses).toContain("validating");
      expect(allTab!.statuses).toContain("needs_fixup");
    });
  });
});
