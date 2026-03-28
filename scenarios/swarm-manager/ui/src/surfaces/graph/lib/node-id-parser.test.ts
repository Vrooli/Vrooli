import { describe, it, expect } from "vitest";
import { parseNodeId } from "./node-id-parser";

describe("parseNodeId", () => {
  // Server-side projection format (prefixed).
  it("parses backlog-item/{kind}/{name}", () => {
    const result = parseNodeId("backlog-item/execute/my-feature");
    expect(result).toEqual({
      entityType: "backlog",
      identifier: "execute/my-feature",
      kind: "execute",
      name: "my-feature",
    });
  });

  it("parses backlog-item with nested name", () => {
    const result = parseNodeId("backlog-item/research/deep/nested/name");
    expect(result).toEqual({
      entityType: "backlog",
      identifier: "research/deep/nested/name",
      kind: "research",
      name: "deep/nested/name",
    });
  });

  it("parses scenario/{name}", () => {
    const result = parseNodeId("scenario/swarm-manager");
    expect(result).toEqual({
      entityType: "scenario",
      identifier: "swarm-manager",
      name: "swarm-manager",
    });
  });

  it("parses execution-record/{id}", () => {
    const result = parseNodeId("execution-record/abc-123");
    expect(result).toEqual({
      entityType: "execution",
      identifier: "abc-123",
      name: "abc-123",
    });
  });

  it("parses agent-run/{runId} (prefixed)", () => {
    const result = parseNodeId("agent-run/run-456");
    expect(result).toEqual({
      entityType: "agent-run",
      identifier: "run-456",
      name: "run-456",
    });
  });

  it("parses capture/{id}", () => {
    const result = parseNodeId("capture/cap-789");
    expect(result).toEqual({
      entityType: "capture",
      identifier: "cap-789",
      name: "cap-789",
    });
  });

  it("parses initiative/{name}", () => {
    const result = parseNodeId("initiative/graph-workspace");
    expect(result).toEqual({
      entityType: "initiative",
      identifier: "graph-workspace",
      name: "graph-workspace",
    });
  });

  // Legacy format (from client-side assembler).
  it("parses legacy backlog format {kind}/{name}", () => {
    const result = parseNodeId("execute/some-feature");
    expect(result).toEqual({
      entityType: "backlog",
      identifier: "execute/some-feature",
      kind: "execute",
      name: "some-feature",
    });
  });

  it("parses legacy execution/{id}", () => {
    const result = parseNodeId("execution/xyz-789");
    expect(result).toEqual({
      entityType: "execution",
      identifier: "xyz-789",
    });
  });

  it("parses legacy scenario/{name}", () => {
    const result = parseNodeId("scenario/my-scenario");
    expect(result).toEqual({
      entityType: "scenario",
      identifier: "my-scenario",
      name: "my-scenario",
    });
  });

  // Edge cases.
  it("returns null for empty string", () => {
    expect(parseNodeId("")).toBeNull();
  });

  it("returns null for unrecognized format", () => {
    expect(parseNodeId("unknown-prefix/foo")).toBeNull();
  });

  it("returns null for prefix with no identifier", () => {
    expect(parseNodeId("scenario/")).toBeNull();
  });

  it("returns null for backlog-item with no name after kind", () => {
    expect(parseNodeId("backlog-item/execute")).toBeNull();
  });
});
