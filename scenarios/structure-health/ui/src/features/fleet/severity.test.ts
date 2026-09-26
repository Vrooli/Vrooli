import { describe, expect, it } from "vitest";

import { severityChipClass } from "./severity";

describe("severityChipClass", () => {
  it("maps known severities to distinct token-driven chip classes", () => {
    const error = severityChipClass("error");
    const warning = severityChipClass("warning");
    const info = severityChipClass("info");

    expect(error).toContain("app-danger");
    expect(warning).toContain("app-warning");
    expect(info).toContain("app-info");
    expect(new Set([error, warning, info]).size).toBe(3);
  });

  it("is case-insensitive", () => {
    expect(severityChipClass("ERROR")).toBe(severityChipClass("error"));
    expect(severityChipClass("Warning")).toBe(severityChipClass("warning"));
  });

  it("falls back to a muted class for unknown severities", () => {
    const fallback = severityChipClass("critical");
    expect(fallback).toContain("app-muted-foreground");
    expect(fallback).toBe(severityChipClass(""));
  });
});
