import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { distinctVariables, renderSnippet } from "./snippetVars";

describe("renderSnippet", () => {
  it("returns a body with no variables unchanged", () => {
    expect(renderSnippet("Demand exact evidence.", {})).toBe("Demand exact evidence.");
  });

  it("substitutes a supplied lowercase name", () => {
    expect(renderSnippet("Check {{scenario}} first.", { scenario: "web-console" }))
      .toBe("Check web-console first.");
  });

  it("leaves an absent name verbatim", () => {
    expect(renderSnippet("Check {{scenario}} first.", {}))
      .toBe("Check {{scenario}} first.");
  });

  it("substitutes an explicitly supplied blank", () => {
    expect(renderSnippet("Check {{scenario}} first.", { scenario: "" }))
      .toBe("Check  first.");
  });

  it("replaces every occurrence", () => {
    expect(renderSnippet("{{a}} and {{a}}", { a: "x" })).toBe("x and x");
  });

  it.each(["{{A}}", "{{1x}}", "{{ x }}"])("treats %s as literal text", (body) => {
    expect(renderSnippet(body, { A: "no", "1x": "no", x: "no" })).toBe(body);
  });
});

describe("distinctVariables", () => {
  it("deduplicates in first-appearance order", () => {
    expect(distinctVariables("{{b}} {{a}} {{b}}"))
      .toEqual(["b", "a"]);
  });
});

describe("module boundary", () => {
  it("keeps the implementation dependency-free", () => {
    const sourcePath = resolve(process.cwd(), "src/lib/snippetVars.ts");
    const source = readFileSync(sourcePath, "utf8");
    expect(source).not.toMatch(/^\s*import(?:\s|\{)/m);
  });
});
