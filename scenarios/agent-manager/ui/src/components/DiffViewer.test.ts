import { describe, it, expect } from "vitest";
import { parseHunks } from "./DiffViewer";

describe("parseHunks", () => {
  it("returns empty array for empty string", () => {
    expect(parseHunks("")).toEqual([]);
  });

  it("returns empty array for undefined-ish input", () => {
    expect(parseHunks("")).toEqual([]);
  });

  it("parses a single hunk with adds and removes", () => {
    const patch = [
      "@@ -1,3 +1,4 @@",
      " line1",
      "-line2",
      "+line2-modified",
      "+line2b-new",
      " line3",
    ].join("\n");

    const hunks = parseHunks(patch);
    expect(hunks).toHaveLength(1);
    expect(hunks[0].oldStart).toBe(1);
    expect(hunks[0].newStart).toBe(1);
    expect(hunks[0].lines).toHaveLength(5);

    // Context line
    expect(hunks[0].lines[0]).toEqual({
      type: "context",
      content: "line1",
      oldLine: 1,
      newLine: 1,
    });

    // Remove line
    expect(hunks[0].lines[1]).toEqual({
      type: "remove",
      content: "line2",
      oldLine: 2,
    });

    // Add lines
    expect(hunks[0].lines[2]).toEqual({
      type: "add",
      content: "line2-modified",
      newLine: 2,
    });
    expect(hunks[0].lines[3]).toEqual({
      type: "add",
      content: "line2b-new",
      newLine: 3,
    });

    // Context line (line numbers advanced)
    expect(hunks[0].lines[4]).toEqual({
      type: "context",
      content: "line3",
      oldLine: 3,
      newLine: 4,
    });
  });

  it("parses multiple hunks", () => {
    const patch = [
      "@@ -1,2 +1,2 @@",
      "-old1",
      "+new1",
      " same",
      "@@ -10,2 +10,2 @@",
      "-old10",
      "+new10",
      " same10",
    ].join("\n");

    const hunks = parseHunks(patch);
    expect(hunks).toHaveLength(2);
    expect(hunks[0].oldStart).toBe(1);
    expect(hunks[0].newStart).toBe(1);
    expect(hunks[1].oldStart).toBe(10);
    expect(hunks[1].newStart).toBe(10);
  });

  it("parses hunk header with function context", () => {
    const patch = "@@ -5,3 +5,4 @@ function foo() {\n+  bar();\n";
    const hunks = parseHunks(patch);
    expect(hunks).toHaveLength(1);
    expect(hunks[0].context).toBe("function foo() {");
  });

  it("handles hunk header with no lines following", () => {
    const patch = "@@ -1,0 +1,0 @@";
    const hunks = parseHunks(patch);
    expect(hunks).toHaveLength(1);
    expect(hunks[0].lines).toEqual([]);
  });

  it("handles 'No newline at end of file' marker", () => {
    const patch = [
      "@@ -1,1 +1,1 @@",
      "-old",
      "\\ No newline at end of file",
      "+new",
    ].join("\n");

    const hunks = parseHunks(patch);
    expect(hunks[0].lines[1]).toEqual({
      type: "no-newline",
      content: "\\ No newline at end of file",
    });
  });

  it("handles lines starting with + or - that are diff content", () => {
    // A file with content "+1" and "-1" — these are still diff add/remove lines
    const patch = [
      "@@ -1,1 +1,1 @@",
      "-+1",
      "+-1",
    ].join("\n");

    const hunks = parseHunks(patch);
    // "-+1" means removing a line whose content is "+1"
    expect(hunks[0].lines[0]).toEqual({
      type: "remove",
      content: "+1",
      oldLine: 1,
    });
    // "+-1" means adding a line whose content is "-1"
    expect(hunks[0].lines[1]).toEqual({
      type: "add",
      content: "-1",
      newLine: 1,
    });
  });

  it("ignores lines before first hunk header", () => {
    const patch = [
      "diff --git a/foo.ts b/foo.ts",
      "index abc123..def456 100644",
      "--- a/foo.ts",
      "+++ b/foo.ts",
      "@@ -1,1 +1,1 @@",
      "-old",
      "+new",
    ].join("\n");

    const hunks = parseHunks(patch);
    expect(hunks).toHaveLength(1);
    expect(hunks[0].lines).toHaveLength(2);
  });
});
