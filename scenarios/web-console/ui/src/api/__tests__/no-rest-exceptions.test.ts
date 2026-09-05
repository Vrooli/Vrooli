import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// Regression guard for the template's REST-exception contract.
//
// Every UI -> API call goes through Connect-RPC. The only sanctioned
// exceptions are the files allowlisted below, each tied to one
// of the template's enumerated RESTReason values:
//
//   health.ts   -> RESTReasonOpsProbe
//   uploads.ts  -> RESTReasonMultipartUpload
//   ttsHook.ts  -> RESTReasonHostHookGlue (Claude project-settings hook
//                  routing / playback diagnostics — a deliberately tiny
//                  web-console-internal surface that never crosses scenario
//                  boundaries; all audio synthesis flows through Connect
//                  against audio-tools)
// See docs/internal/SEAMS.md for the registry. Adding a fourth REST
// surface requires picking another enumerated RESTReason and updating
// SEAMS.md AND this allowlist in the same change.

const THIS_FILE = fileURLToPath(import.meta.url);
const API_DIR = resolve(THIS_FILE, "..", "..");

const ALLOWLIST = new Set(["health.ts", "uploads.ts", "ttsHook.ts"]);

function listApiFiles(dir: string): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) continue;
    if (!/\.ts$/.test(entry)) continue;
    if (/\.test\.ts$/.test(entry)) continue;
    out.push(full);
  }
  return out;
}

describe("REST exceptions (UI api/)", () => {
  it("no fetch( outside the two sanctioned REST exception files", () => {
    const files = listApiFiles(API_DIR);
    const offenders: { file: string; line: number; text: string }[] = [];

    for (const file of files) {
      const basename = file.split("/").pop() ?? file;
      if (ALLOWLIST.has(basename)) continue;
      const text = readFileSync(file, "utf8");
      const lines = text.split("\n");
      lines.forEach((line, idx) => {
        if (line.includes("fetch(")) {
          offenders.push({ file: basename, line: idx + 1, text: line.trim() });
        }
      });
    }

    expect(
      offenders,
      `Found fetch( in non-sanctioned api/ files. Either route through Connect-RPC ` +
        `or, if the call genuinely matches one of the template's RESTReason values, ` +
        `document the exception in docs/internal/SEAMS.md and extend the allowlist ` +
        `here. Offenders:\n` +
        offenders.map((o) => `  ${o.file}:${o.line}  ${o.text}`).join("\n"),
    ).toEqual([]);
  });

  it("each allowlisted REST exception file actually contains fetch(", () => {
    // Sanity check: the allowlist should track real usage, not stale
    // entries. If a file is allowlisted but no longer uses fetch(),
    // remove it from ALLOWLIST.
    for (const name of ALLOWLIST) {
      const path = resolve(API_DIR, name);
      const text = readFileSync(path, "utf8");
      expect(text, `${name} is allowlisted but contains no fetch( — remove from ALLOWLIST`).toMatch(/fetch\(/);
    }
  });
});
