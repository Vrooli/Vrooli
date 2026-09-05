import { describe, expect, it } from "vitest";
import { nodeIdForSessionArtifact } from "./session-artifact-routing";

describe("nodeIdForSessionArtifact", () => {
  it("maps openable artifact refs to graph node ids", () => {
    expect(nodeIdForSessionArtifact({ artifactType: "backlog_item", entityRef: "scenario-a/item-b" })).toBe("backlog-item/scenario-a/item-b");
    expect(nodeIdForSessionArtifact({ artifactType: "capture", entityRef: "cap-1" })).toBe("capture/cap-1");
    expect(nodeIdForSessionArtifact({ artifactType: "agent_activity", entityRef: "act-1" })).toBe("agent-activity/act-1");
  });

  it("returns null for unsupported or malformed refs", () => {
    expect(nodeIdForSessionArtifact({ artifactType: "file", entityRef: "notes.md" })).toBeNull();
    expect(nodeIdForSessionArtifact({ artifactType: "backlog_item", entityRef: "missing-slash" })).toBeNull();
    expect(nodeIdForSessionArtifact({ artifactType: "milestone", entityRef: "quality-gates" })).toBeNull();
  });
});
