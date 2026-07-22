import { beforeEach, describe, expect, it } from "vitest";
import type { SessionContextOption } from "./session-context-refs";
import { compatibleSessionKindsForContextType } from "./session-context-config";
import {
  clearStagedContextForSession,
  mergeContextOptions,
  peekStagedContextForSession,
  stageContextForSession,
} from "./pending-session-context";

const goal: SessionContextOption = {
  type: "goal",
  ref: "release-control",
  title: "Release Control",
};

const scenario: SessionContextOption = {
  type: "scenario",
  ref: "swarm-manager",
  title: "swarm-manager",
};

describe("pending session context", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stages context by session without duplicating the same ref [REQ:REQ-P1-010-SESSION-CONTEXT]", () => {
    stageContextForSession("sess_1", goal);
    stageContextForSession("sess_1", goal);

    expect(peekStagedContextForSession("sess_1")).toEqual([goal]);

    clearStagedContextForSession("sess_1", [goal]);
    expect(peekStagedContextForSession("sess_1")).toEqual([]);
  });

  it("merges compatible context and rejects incompatible context visibly [REQ:REQ-P1-010-SESSION-CONTEXT]", () => {
    const result = mergeContextOptions([goal], [goal, scenario], "swarm_operations");

    expect(result.items).toEqual([goal]);
    expect(result.applied).toEqual([goal]);
    expect(result.rejected).toHaveLength(1);
    expect(result.rejected[0]?.reason).toMatch(/not allowed/);
  });

  it("derives compatible session kinds from the shared kind policy [REQ:REQ-P1-010-SESSION-CONTEXT]", () => {
    expect(compatibleSessionKindsForContextType("scenario")).toEqual(["meta_orchestration"]);
    expect(compatibleSessionKindsForContextType("operating_mode")).toEqual([]);
    expect(compatibleSessionKindsForContextType("session")).toEqual(["meta_orchestration", "swarm_operations"]);
  });
});
