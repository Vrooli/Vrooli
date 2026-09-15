import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const styles = readFileSync(resolve(process.cwd(), "src/styles.css"), "utf8");

describe("GCT-MOBILE stylesheet contract", () => {
  it("keeps the iframe-safe root height chain and bounded shell", () => {
    expect(styles).toMatch(/html,\s*body,\s*#root\s*\{[^}]*height:\s*100%/s);
    expect(styles).toMatch(/\.gct-mobile-shell\s*\{[^}]*min-height:\s*100%/s);
    expect(styles).toMatch(/\.gct-mobile-content\s*\{[^}]*overflow:\s*hidden/s);
  });
});
