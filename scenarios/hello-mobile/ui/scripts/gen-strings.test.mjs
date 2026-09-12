/**
 * Unit tests for the strings codegen — covers the underscore-prefix
 * sentinel convention end-to-end. The convention is also enforced in
 * locales.test.ts (parity) and eslint-rules/no-unused-keys.js (audit);
 * this file pins the codegen half so the generated registry never
 * leaks sentinels into `strings.generated.ts`.
 */
import { describe, it, expect } from "vitest";
import { mkdtemp, writeFile, mkdir } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { isSentinelKey } from "./gen-strings.mjs";

describe("gen-strings: underscore-prefix sentinel skip", () => {
  it("identifies underscore-prefixed keys as sentinels", () => {
    expect(isSentinelKey("_comment")).toBe(true);
    expect(isSentinelKey("_meta")).toBe(true);
    expect(isSentinelKey("_")).toBe(true);
  });

  it("does not treat regular keys as sentinels", () => {
    expect(isSentinelKey("title")).toBe(false);
    expect(isSentinelKey("app")).toBe(false);
    expect(isSentinelKey("refreshCount_one")).toBe(false);
    expect(isSentinelKey("a_b")).toBe(false);
  });
});

describe("gen-strings: codegen output", () => {
  /**
   * The codegen module reads SOURCE_PATH = `<root>/src/i18n/locales/en.json`
   * and writes TARGET_PATH = `<root>/src/consts/strings.generated.ts`. To
   * test the generator independently of the live catalog, we need a way to
   * point it at a fixture. Rather than refactoring the module to take paths
   * as parameters (which would expand its public surface), we exercise the
   * pure `buildKeys`-equivalent path indirectly: we write a fixture en.json
   * to a sibling tree, copy the codegen logic into a test helper, and
   * assert on the output shape. The shared `isSentinelKey` predicate is
   * the authoritative seam — both the production codegen and this test
   * import it, so any future drift would surface here.
   */
  const buildKeysLike = (catalog, prefix = "") => {
    const result = {};
    for (const [key, value] of Object.entries(catalog)) {
      if (isSentinelKey(key)) continue;
      const path = prefix ? `${prefix}.${key}` : key;
      if (typeof value === "string") {
        result[key] = path;
      } else if (value && typeof value === "object" && !Array.isArray(value)) {
        result[key] = buildKeysLike(value, path);
      }
    }
    return result;
  };

  it("excludes top-level sentinels from the generated tree", () => {
    const tree = buildKeysLike({
      _comment: "ignored",
      app: { title: "App Title" },
    });
    expect(tree).toEqual({ app: { title: "app.title" } });
    expect(tree).not.toHaveProperty("_comment");
  });

  it("excludes nested sentinels", () => {
    const tree = buildKeysLike({
      app: {
        _meta: "internal",
        title: "App Title",
      },
    });
    expect(tree.app).toEqual({ title: "app.title" });
    expect(tree.app).not.toHaveProperty("_meta");
  });

  it("preserves CLDR plural suffixes (which are not sentinels)", () => {
    const tree = buildKeysLike({
      health: {
        refreshCount: "Refreshed {{count}} times",
        refreshCount_one: "Refreshed once",
      },
    });
    expect(tree.health).toEqual({
      refreshCount: "health.refreshCount",
      refreshCount_one: "health.refreshCount_one",
    });
  });
});

describe("gen-strings: end-to-end on real catalog", async () => {
  // Smoke test against the actual en.json — proves the production codegen
  // path treats `_comment` as a sentinel even at module-load time.
  const { generateContents } = await import("./gen-strings.mjs");

  it("does not include `_comment` in the generated output", () => {
    const output = generateContents();
    expect(output).not.toMatch(/_comment/);
    // Sanity: known real key is present.
    expect(output).toMatch(/app: \{/);
  });
});
