import { describe, it, expect } from "vitest";
import {
  BLOCKING_AGENT_ACTIVITY_STATUSES,
  EXECUTING_AGENT_ACTIVITY_STATUSES,
  getAgentActivityLabel,
  getAgentActivityTone,
  isAgentActivityBlocking,
  isAgentActivityExecuting,
} from "./agent-activity-utils";

describe("agent-activity-utils", () => {
  describe("isAgentActivityExecuting", () => {
    it("returns true for pending/starting/running", () => {
      expect(isAgentActivityExecuting("pending")).toBe(true);
      expect(isAgentActivityExecuting("starting")).toBe(true);
      expect(isAgentActivityExecuting("running")).toBe(true);
    });

    it("returns false for review-waiting and terminal/idle statuses", () => {
      expect(isAgentActivityExecuting("needs_review")).toBe(false);
      expect(isAgentActivityExecuting("complete")).toBe(false);
      expect(isAgentActivityExecuting("failed")).toBe(false);
      expect(isAgentActivityExecuting("cancelled")).toBe(false);
      expect(isAgentActivityExecuting("unspecified")).toBe(false);
    });

    it("returns false for null/undefined", () => {
      expect(isAgentActivityExecuting(null)).toBe(false);
      expect(isAgentActivityExecuting(undefined)).toBe(false);
    });

    it("keeps the exported executing-set in sync", () => {
      // Regression guard: if someone adds a status to EXECUTING_AGENT_ACTIVITY_STATUSES,
      // the predicate should agree.
      for (const status of EXECUTING_AGENT_ACTIVITY_STATUSES) {
        expect(isAgentActivityExecuting(status)).toBe(true);
      }
    });
  });

  describe("isAgentActivityBlocking", () => {
    it("returns true for running states plus needs_review", () => {
      expect(isAgentActivityBlocking("pending")).toBe(true);
      expect(isAgentActivityBlocking("starting")).toBe(true);
      expect(isAgentActivityBlocking("running")).toBe(true);
      expect(isAgentActivityBlocking("needs_review")).toBe(true);
    });

    it("returns false for terminal/idle statuses", () => {
      expect(isAgentActivityBlocking("complete")).toBe(false);
      expect(isAgentActivityBlocking("failed")).toBe(false);
      expect(isAgentActivityBlocking("cancelled")).toBe(false);
      expect(isAgentActivityBlocking("unspecified")).toBe(false);
    });

    it("keeps the exported blocking-set in sync", () => {
      for (const status of BLOCKING_AGENT_ACTIVITY_STATUSES) {
        expect(isAgentActivityBlocking(status)).toBe(true);
      }
    });
  });

  describe("getAgentActivityLabel", () => {
    it("maps known purposes to human labels", () => {
      expect(getAgentActivityLabel("workshop")).toBe("Workshopping");
      expect(getAgentActivityLabel("finalize")).toBe("Finalizing");
      expect(getAgentActivityLabel("review")).toBe("Reviewing");
      expect(getAgentActivityLabel("fixup")).toBe("Fixing up");
      expect(getAgentActivityLabel("followup")).toBe("Following up");
      expect(getAgentActivityLabel("research")).toBe("Researching");
      expect(getAgentActivityLabel("spec_sync")).toBe("Syncing spec");
    });

    it("falls back to a generic label for null/undefined", () => {
      expect(getAgentActivityLabel(null)).toBe("Agent running");
      expect(getAgentActivityLabel(undefined)).toBe("Agent running");
    });
  });

  describe("getAgentActivityTone", () => {
    it("returns 'needs-review' only for needs_review", () => {
      expect(getAgentActivityTone("needs_review")).toBe("needs-review");
    });

    it("returns 'busy' for active work statuses", () => {
      expect(getAgentActivityTone("pending")).toBe("busy");
      expect(getAgentActivityTone("starting")).toBe("busy");
      expect(getAgentActivityTone("running")).toBe("busy");
    });
  });
});
