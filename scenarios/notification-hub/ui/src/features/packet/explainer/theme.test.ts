import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

import { pageColorScheme, pageTheme } from "./theme";

// Vitest stubs CSS imports, so the stylesheet is read from disk.
const kitCss = readFileSync(join(process.cwd(), "src/features/packet/explainer/kit/kit.css"), "utf8");
// The API's packet check reads the same fixtures (page_theme_internal_test.go),
// so the frame and the check agree on every page. The name before "--" is the
// scheme the page was written for.
const fixtureDir = join(process.cwd(), "..", "api", "internal", "hub", "testdata", "page-schemes");
const fixtures = readdirSync(fixtureDir).filter((name) => name.endsWith(".html"));

describe("pageColorScheme", () => {
  it("has fixtures to read", () => {
    expect(fixtures.length).toBeGreaterThanOrEqual(8);
  });

  it.each(fixtures)("reads %s as the scheme its colours were written for", (name) => {
    const html = readFileSync(join(fixtureDir, name), "utf8");
    expect(pageColorScheme(html, kitCss)).toBe(name.split("--")[0]);
  });
});

describe("pageTheme", () => {
  it("follows the Hub for a page that works in both, and keeps a one-scheme page in its own", () => {
    expect(pageTheme("both", "dark")).toBe("dark");
    expect(pageTheme("both", "light")).toBe("light");
    expect(pageTheme("light", "dark")).toBe("light");
    expect(pageTheme("dark", "light")).toBe("dark");
  });
});
