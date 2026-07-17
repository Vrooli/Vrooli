import { describe, expect, it } from "vitest";
import {
  capabilityLabel,
  humanizeRunStrategy,
  humanizeTargetKind,
  parseAcceptanceCriteria,
  serializeAcceptanceCriteria,
} from "./utils";

describe("humanizeTargetKind", () => {
  it("humanizes the registered target kinds", () => {
    expect(humanizeTargetKind("backlog-item")).toBe("Backlog item");
    expect(humanizeTargetKind("initiative")).toBe("Initiative");
    expect(humanizeTargetKind("plan-execution")).toBe("Plan execution");
    expect(humanizeTargetKind("scenario")).toBe("Scenario");
  });
  it("falls back to the raw token for unknown (including retired) values", () => {
    expect(humanizeTargetKind("project")).toBe("project");
    expect(humanizeTargetKind("retired-kind")).toBe("retired-kind");
  });
  it("renders an em-dash for the empty string", () => {
    expect(humanizeTargetKind("")).toBe("—");
  });
});

describe("humanizeRunStrategy", () => {
  it("humanizes registered run strategies", () => {
    expect(humanizeRunStrategy("single_phase_run")).toBe("Single phase run");
    expect(humanizeRunStrategy("sequential_handoff")).toBe("Sequential handoff");
    expect(humanizeRunStrategy("operator_gated_loop")).toBe("Operator-gated loop");
  });
  it("falls back to the raw token for unknown values", () => {
    expect(humanizeRunStrategy("some_new_strategy")).toBe("some_new_strategy");
  });
});

describe("parseAcceptanceCriteria", () => {
  it("returns an empty array for empty input", () => {
    expect(parseAcceptanceCriteria("")).toEqual([]);
  });
  it("trims whitespace and drops empty lines", () => {
    expect(parseAcceptanceCriteria("  All tests pass  \n\n  Docs updated  \n")).toEqual([
      "All tests pass",
      "Docs updated",
    ]);
  });
  it("treats whitespace-only input as empty", () => {
    expect(parseAcceptanceCriteria("   \n\t\n   ")).toEqual([]);
  });
  it("handles CRLF line endings", () => {
    expect(parseAcceptanceCriteria("First\r\nSecond\r\nThird")).toEqual([
      "First",
      "Second",
      "Third",
    ]);
  });
});

describe("serializeAcceptanceCriteria", () => {
  it("joins with newlines", () => {
    expect(serializeAcceptanceCriteria(["a", "b", "c"])).toBe("a\nb\nc");
  });
  it("returns an empty string for an empty array", () => {
    expect(serializeAcceptanceCriteria([])).toBe("");
  });
  it("round-trips through parseAcceptanceCriteria", () => {
    const original = ["All tests pass", "No new lint errors", "Docs updated"];
    expect(parseAcceptanceCriteria(serializeAcceptanceCriteria(original))).toEqual(original);
  });
});

describe("capabilityLabel", () => {
  it("labels every known capability flag", () => {
    expect(capabilityLabel("supportsPhases")).toBe("Phase graph");
    expect(capabilityLabel("canStartPhases")).toBe("Phase start controls");
    expect(capabilityLabel("canCompleteItems")).toBe("Mark items complete from rounds");
    expect(capabilityLabel("canApplyBacklogSyncProposals")).toBe("Apply backlog sync proposals");
    expect(capabilityLabel("requiresAcceptanceCriteria")).toBe("Requires acceptance criteria");
    expect(capabilityLabel("supportsArtifacts")).toBe("Phase artifacts");
    expect(capabilityLabel("supportsHandoffs")).toBe("Round handoffs");
  });
});
