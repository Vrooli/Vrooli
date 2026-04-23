import { describe, it, expect } from "vitest";
import {
  ACTIVE_AGENT_ACTIVITY_STATUSES,
  getAgentActivityLabel,
  getAgentActivityTone,
  isAgentActivityActive,
} from "./agent-activity-utils";

describe("agent-activity-utils", () => {
  describe("isAgentActivityActive", () => {
    it("returns true for pending/starting/running/needs_review", () => {
      expect(isAgentActivityActive("pending")).toBe(true);
      expect(isAgentActivityActive("starting")).toBe(true);
      expect(isAgentActivityActive("running")).toBe(true);
      expect(isAgentActivityActive("needs_review")).toBe(true);
    });

    it("returns false for terminal/idle statuses", () => {
      expect(isAgentActivityActive("complete")).toBe(false);
      expect(isAgentActivityActive("failed")).toBe(false);
      expect(isAgentActivityActive("cancelled")).toBe(false);
      expect(isAgentActivityActive("unspecified")).toBe(false);
    });

    it("returns false for null/undefined", () => {
      expect(isAgentActivityActive(null)).toBe(false);
      expect(isAgentActivityActive(undefined)).toBe(false);
    });

    it("keeps the exported active-set in sync", () => {
      // Regression guard: if someone adds a status to ACTIVE_AGENT_ACTIVITY_STATUSES,
      // the predicate should agree.
      for (const status of ACTIVE_AGENT_ACTIVITY_STATUSES) {
        expect(isAgentActivityActive(status)).toBe(true);
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
