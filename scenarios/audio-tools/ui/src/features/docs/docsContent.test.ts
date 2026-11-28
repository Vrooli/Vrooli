/**
 * Tests for docsContent.ts
 *
 * The module uses lazy Vite `import.meta.glob` imports. Vitest resolves these
 * globs so each actual scenario Markdown document can be loaded on demand.
 */
import { describe, it, expect } from "vitest";

import { getDocContent, listDocPaths } from "./docsContent";

describe("listDocPaths", () => {
  it("returns the bundled, scenario-root-relative markdown paths", () => {
    const paths = listDocPaths();
    expect(Array.isArray(paths)).toBe(true);
    expect(paths.length).toBeGreaterThan(0);
    // Keys are normalised to scenario-root-relative paths (prefix stripped).
    expect(paths).toContain("PRD.md");
    for (const p of paths) {
      expect(p.startsWith("../")).toBe(false);
    }
  });

  it("includes entries from the docs/ subtree", () => {
    const paths = listDocPaths();
    expect(paths.some((p) => p.startsWith("docs/"))).toBe(true);
  });
});

describe("getDocContent", () => {
  it("returns the raw markdown for a bundled path", async () => {
    const content = await getDocContent("PRD.md");
    expect(typeof content).toBe("string");
    expect(content!.length).toBeGreaterThan(0);
  });

  it("loads the content for every advertised path", async () => {
    const contents = await Promise.all(listDocPaths().map((path) => getDocContent(path)));
    expect(contents).toHaveLength(listDocPaths().length);
    expect(contents.every((content) => typeof content === "string" && content.length > 0)).toBe(true);
  });

  it("returns null for an unbundled or empty path", async () => {
    await expect(getDocContent("nonexistent/path.md")).resolves.toBeNull();
    await expect(getDocContent("")).resolves.toBeNull();
  });
});
