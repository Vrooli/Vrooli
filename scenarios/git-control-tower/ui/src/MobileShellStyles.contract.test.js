import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const styles = readFileSync(resolve(process.cwd(), "src/styles.css"), "utf8");

describe("GCT-MOBILE stylesheet contract", () => {
  it("sizes the shell from the measured viewport and locks the page (iOS-safe)", () => {
    // The shared environment measures the drawable canvas. Installed iOS can
    // report innerHeight/dvh 793 while the measured large viewport is 852;
    // the roots and shell must use that same frame to avoid clipping the nav.
    expect(styles).toMatch(/\.gct-mobile-shell\s*\{[^}]*height:\s*var\(--gct-app-height/s);
    // The page itself must be scroll-locked so iOS overscroll cannot drag the
    // whole shell (and the bottom nav) out of position.
    expect(styles).toMatch(/body\s*\{[^}]*position:\s*fixed/s);
    expect(styles).toMatch(/overscroll-behavior:\s*none/s);
    // The active panel still owns scrolling, not the page.
    expect(styles).toMatch(/\.gct-mobile-content\s*\{[^}]*overflow:\s*hidden/s);
  });
});
