import { describe, expect, it } from "vitest";
import { humanizeRunStrategy, humanizeScopeKind } from "./utils";

describe("humanizeScopeKind", () => {
  it("humanizes registered scope kinds", () => {
    expect(humanizeScopeKind("backlog_item")).toBe("Backlog item");
    expect(humanizeScopeKind("initiative")).toBe("Initiative");
  });
  it("falls back to the raw token for unknown values", () => {
    expect(humanizeScopeKind("project")).toBe("project");
  });
  it("renders an em-dash for the empty string", () => {
    expect(humanizeScopeKind("")).toBe("—");
  });
});

describe("humanizeRunStrategy", () => {
  it("humanizes registered run strategies", () => {
    expect(humanizeRunStrategy("existing_item_flow")).toBe("Existing item flow");
    expect(humanizeRunStrategy("single_phase_run")).toBe("Single phase run");
    expect(humanizeRunStrategy("sequential_handoff")).toBe("Sequential handoff");
    expect(humanizeRunStrategy("operator_gated_loop")).toBe("Operator-gated loop");
  });
  it("falls back to the raw token for unknown values", () => {
    expect(humanizeRunStrategy("some_new_strategy")).toBe("some_new_strategy");
  });
});
