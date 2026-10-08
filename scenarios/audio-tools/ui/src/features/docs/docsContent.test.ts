/**
 * Tests for docsContent.ts
 *
 * The module uses Vite's `import.meta.glob(..., { eager: true })` to bundle
 * the scenario's markdown at build time. Vitest resolves these globs, so the
 * `docs` map is populated with the real scenario docs (PRD.md, README.md,
 * and the docs/** tree). We assert the public surface against that.
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
  it("returns the raw markdown for a bundled path", () => {
    const content = getDocContent("PRD.md");
    expect(typeof content).toBe("string");
    expect(content!.length).toBeGreaterThan(0);
  });

  it("returns the same content listDocPaths advertises for every key", () => {
    for (const p of listDocPaths()) {
      expect(typeof getDocContent(p)).toBe("string");
    }
  });

  it("returns null for an unbundled or empty path", () => {
    expect(getDocContent("nonexistent/path.md")).toBeNull();
    expect(getDocContent("")).toBeNull();
  });
});
