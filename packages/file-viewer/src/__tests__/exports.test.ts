// @vitest-environment node
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

/**
 * Tailwind loads `tailwind.config.ts` through jiti, which resolves imports with
 * the `require` condition; a subpath that declares only `import` is then "not
 * defined by exports". Every conditional export must therefore also carry
 * `default`, pointing at the same file.
 */
const manifest = JSON.parse(readFileSync("package.json", "utf8")) as { exports: Record<string, string | Record<string, string>> };
const conditional = Object.entries(manifest.exports).filter((entry): entry is [string, Record<string, string>] => typeof entry[1] === "object");

describe("package exports", () => {
  it("has conditional exports to check", () => { expect(conditional.length).toBeGreaterThan(5); });

  it.each(conditional)("%s resolves for require-condition loaders as well as import", (_subpath, conditions) => {
    expect(conditions.default).toBeDefined();
    expect(conditions.default).toBe(conditions.import);
  });
});
