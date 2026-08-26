import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const componentsDir = path.dirname(fileURLToPath(import.meta.url));

/**
 * Components must deliver their stylesheet through `useComponentStyles`, which
 * injects one real `<style>` element per unique id into `document.head`.
 * Rendering `<style dangerouslySetInnerHTML>` inline puts one byte-identical
 * copy in the DOM per mounted instance.
 *
 * Exemptions: `{ file, reason }` per entry, and only for a case that genuinely
 * cannot use the hook. There are none today; adding one requires a reason that
 * explains why the hook does not work, not that it was inconvenient.
 */
const EXEMPTIONS: Array<{ file: string; reason: string }> = [];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return sourceFiles(full);
    if (!entry.isFile() || !/\.tsx?$/.test(entry.name)) return [];
    if (/\.test\.tsx?$/.test(entry.name)) return [];
    return [full];
  });
}

// Matches `<style ... dangerouslySetInnerHTML={{ __html: ... }} ... >` across
// line breaks. Deliberately narrow: `dangerouslySetInnerHTML` on a non-style
// element (rendered markdown HTML, a Mermaid SVG) is a different concern and is
// not what this guard is about.
const INLINE_STYLE_TAG = /<style\b[^>]*?dangerouslySetInnerHTML/s;

describe("component stylesheet delivery", () => {
  const files = sourceFiles(componentsDir);
  const exempt = new Set(EXEMPTIONS.map((entry) => entry.file));

  it("finds component sources to check", () => {
    expect(files.length).toBeGreaterThan(0);
  });

  it("has no component rendering <style dangerouslySetInnerHTML>", () => {
    const offenders = files
      .map((file) => path.relative(componentsDir, file))
      .filter((relative) => !exempt.has(relative))
      .filter((relative) =>
        INLINE_STYLE_TAG.test(readFileSync(path.join(componentsDir, relative), "utf8")),
      );

    expect(offenders).toEqual([]);
  });

  it("only exempts files that still exist", () => {
    const known = new Set(files.map((file) => path.relative(componentsDir, file)));
    expect(EXEMPTIONS.filter((entry) => !known.has(entry.file))).toEqual([]);
  });

  it("requires a reason on every exemption", () => {
    expect(EXEMPTIONS.filter((entry) => entry.reason.trim().length === 0)).toEqual([]);
  });
});

describe("package-backed style delivery", () => {
  it("does not retain legacy vendored foundation copies", () => {
    const legacy = sourceFiles(componentsDir)
      .map((file) => path.relative(componentsDir, file))
      .filter((file) => file.includes("/versions/") && /ControlBase|Pressable/.test(file));
    expect(legacy).toEqual([]);
  });
});
