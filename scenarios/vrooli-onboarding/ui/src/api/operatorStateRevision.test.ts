import { describe, expect, it } from "vitest";
import { operatorStateRevision } from "./operatorStateRevision";

describe("operatorStateRevision", () => {
  it("prefers a durable update timestamp", () => {
    expect(operatorStateRevision({ updatedAt: "  2026-11-21T12:00:00Z  ", version: "1.0.0" })).toBe("2026-11-21T12:00:00Z");
  });

  it("falls back to the schema revision when updatedAt is empty", () => {
    expect(operatorStateRevision({ updatedAt: "", version: "1.0.0" })).toBe("1.0.0");
  });

  it("returns an empty token only when neither revision exists", () => {
    expect(operatorStateRevision(undefined)).toBe("");
    expect(operatorStateRevision({ updatedAt: "  ", version: " " })).toBe("");
  });
});
