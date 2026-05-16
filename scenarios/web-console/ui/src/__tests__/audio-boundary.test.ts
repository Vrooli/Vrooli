import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Audio adoption boundary guard.
 *
 * `src/domains/audio/index.ts` is the single re-export surface for the
 * reusable voice/TTS capability code that will move into the future
 * `scenarios/audio-tools` scenario. Every consumer in the UI must import
 * from `domains/audio` rather than reaching directly into the underlying
 * `hooks/voice/**` or `hooks/tts/**` modules — otherwise the audio-tools
 * adoption can't ship as a single redirect at the boundary file.
 *
 * Three categories of files are allowed to reach into those subtrees:
 *
 * 1. `src/domains/audio/**` itself — the adoption surface, by definition.
 * 2. `src/hooks/voice/**` and `src/hooks/tts/**` — intra-tree imports.
 * 3. Test files (`*.test.ts` / `*.test.tsx`) — they may test the
 *    underlying provider/hook implementations directly.
 *
 * Additionally, README classifies a small set of files as web-console
 * specific (commands.ts, commandParser.ts, audioCues.ts, activity.ts,
 * useVoiceInput.ts, useTextToSpeech.ts). Production consumers of those
 * files are NOT migrated through the boundary because they will stay in
 * web-console after extraction. We allow direct imports of those exact
 * web-console-specific paths.
 */

const THIS_FILE: string = fileURLToPath(import.meta.url);
const SRC_ROOT: string = resolve(THIS_FILE, "..", "..");

const ADOPTION_SURFACE = join(SRC_ROOT, "domains", "audio");
const VOICE_HOOKS = join(SRC_ROOT, "hooks", "voice");
const TTS_HOOKS = join(SRC_ROOT, "hooks", "tts");

/**
 * Imports that are allowed to remain after migration because the README
 * classifies the target as "web-console specific" (stays in web-console
 * after audio-tools extraction). Match the import path suffix.
 */
const WEB_CONSOLE_SPECIFIC_ALLOW = [
  "hooks/voice/commands",
  "hooks/voice/commandParser",
  "hooks/voice/audioCues",
  "hooks/voice/activity",
];

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

function isUnderDir(file: string, dir: string): boolean {
  return file === dir || file.startsWith(dir + "/");
}

function isTestFile(file: string): boolean {
  return /\.(test|spec)\.(ts|tsx)$/.test(file);
}

const IMPORT_RE = /\bfrom\s+["']([^"']+)["']/g;

describe("frontend audio adoption boundary", () => {
  it("no consumer imports from hooks/voice/** or hooks/tts/** outside the boundary", () => {
    const files = walk(SRC_ROOT);
    const violations: { file: string; spec: string }[] = [];

    for (const file of files) {
      // Skip the adoption surface — it's allowed to re-export from the
      // underlying subtrees by design.
      if (isUnderDir(file, ADOPTION_SURFACE)) continue;
      // Skip intra-tree imports inside the voice/tts hook implementations.
      if (isUnderDir(file, VOICE_HOOKS)) continue;
      if (isUnderDir(file, TTS_HOOKS)) continue;
      // Skip test files — they may exercise the underlying implementation.
      if (isTestFile(file)) continue;

      const text = readFileSync(file, "utf8");
      IMPORT_RE.lastIndex = 0;
      let match: RegExpExecArray | null;
      while ((match = IMPORT_RE.exec(text)) !== null) {
        const spec = match[1];
        if (typeof spec !== "string") continue;
        if (!/hooks\/(voice|tts)(\/|$)/.test(spec)) continue;
        if (WEB_CONSOLE_SPECIFIC_ALLOW.some((suffix) => spec.endsWith(suffix))) continue;
        violations.push({ file, spec });
      }
    }

    expect(
      violations,
      "Consumers must import audio capability from `domains/audio`, not " +
        "directly from `hooks/voice/**` or `hooks/tts/**`. Move the symbol " +
        "into `src/domains/audio/index.ts` as a re-export (if the README " +
        "classifies it as reusable), then re-point the import. Offenders:\n" +
        violations.map((v) => `  ${v.file}\n    -> "${v.spec}"`).join("\n"),
    ).toEqual([]);
  });

  it("the adoption surface file exists and re-exports from @audio-tools/embed", () => {
    const indexPath = join(ADOPTION_SURFACE, "index.ts");
    const text = readFileSync(indexPath, "utf8");
    // Post-extraction: the seam re-exports the audio-tools embed package.
    expect(text).toMatch(/from\s+["']@audio-tools\/embed["']/);
  });
});
