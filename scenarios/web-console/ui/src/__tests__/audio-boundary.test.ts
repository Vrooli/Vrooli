import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Audio adoption boundary guard.
 *
 * `src/audio-integration/` is the canonical copy-paste reference for the
 * audio-tools integration surface (Connect client, provider hooks, API
 * wrappers). It is duplicated verbatim from
 * `scenarios/audio-tools/ui/src/audio-integration/` — no cross-scenario
 * import. Every consumer in this UI must import from `audio-integration`
 * rather than reaching across to the audio-tools scenario or the legacy
 * `@audio-tools/embed` package.
 */

const THIS_FILE: string = fileURLToPath(import.meta.url);
const SRC_ROOT: string = resolve(THIS_FILE, "..", "..");

const ADOPTION_SURFACE = join(SRC_ROOT, "audio-integration");

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) {
      if (entry === "node_modules" || entry === "dist") continue;
      walk(full, out);
      continue;
    }
    if (/\.(ts|tsx)$/.test(entry)) out.push(full);
  }
  return out;
}

const IMPORT_RE = /\bfrom\s+["']([^"']+)["']/g;

describe("frontend audio adoption boundary", () => {
  it("the audio-integration folder exists with the expected entry points", () => {
    expect(existsSync(ADOPTION_SURFACE)).toBe(true);
    expect(existsSync(join(ADOPTION_SURFACE, "index.ts"))).toBe(true);
    expect(existsSync(join(ADOPTION_SURFACE, "client.tsx"))).toBe(true);
    expect(existsSync(join(ADOPTION_SURFACE, "api"))).toBe(true);
    expect(existsSync(join(ADOPTION_SURFACE, "hooks"))).toBe(true);
  });

  it("no file imports from @audio-tools/embed (retired)", () => {
    const files = walk(SRC_ROOT);
    const violations: { file: string; spec: string }[] = [];

    for (const file of files) {
      const text = readFileSync(file, "utf8");
      IMPORT_RE.lastIndex = 0;
      let match: RegExpExecArray | null;
      while ((match = IMPORT_RE.exec(text)) !== null) {
        const spec = match[1];
        if (typeof spec !== "string") continue;
        if (spec === "@audio-tools/embed" || spec.startsWith("@audio-tools/embed/")) {
          violations.push({ file, spec });
        }
      }
    }

    expect(
      violations,
      "The @audio-tools/embed package has been retired. Import from " +
        "`audio-integration` (the local copy-paste reference) instead. " +
        "Offenders:\n" +
        violations.map((v) => `  ${v.file}\n    -> "${v.spec}"`).join("\n"),
    ).toEqual([]);
  });
});
