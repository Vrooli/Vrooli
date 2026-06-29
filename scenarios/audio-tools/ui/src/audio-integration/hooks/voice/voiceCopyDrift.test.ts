// Cross-scenario byte-drift guard for the voice helper files that are
// intentionally duplicated across audio-tools/ui, web-console/ui, and
// swarm-manager/ui (per the duplicate-before-extract principle).
//
// Audio-tools is the authoritative copy. This test reads the same file from
// each of the other two scenarios and asserts byte equality. If you change
// any file in the synced set, copy it to the other two scenarios; if you add
// a new shared file, append it to SYNCED_FILES below.
//
// DOC: docs/internal/SEAMS.md#auto-stop-decision (drift-guard subsection).
//
// Lives in audio-tools/ui only — audio-tools is the SSOT scenario for voice
// substrate work and runs first under `vrooli scenario test`. Replicating
// this guard into web-console/swarm-manager would be self-referential noise.

import { describe, it, expect } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";

// Paths are file-relative so the test is robust to the cwd `vitest` is
// launched with (vrooli scenario test sets cwd to the scenario root, raw
// pnpm test sets it to the ui package). Walk up to the repo root, then back
// down into each scenario.
const REPO_ROOT = resolve(__dirname, "../../../../../../../");

// The set of files that MUST be byte-identical across all three scenarios.
// Add to this list when you introduce a new shared voice helper; do NOT
// remove from it without a discussion (it's the only thing keeping the three
// copies in sync until we extract a shared package).
const SYNCED_FILES = [
  "ui/src/audio-integration/hooks/voice/autoStopDecision.ts",
  "ui/src/audio-integration/hooks/voice/autoStopDecision.test.ts",
  "ui/src/audio-integration/hooks/useServerVadStateStore.ts",
] as const;

const SCENARIOS = ["audio-tools", "web-console", "swarm-manager"] as const;

describe("voice helper copy drift across scenarios", () => {
  for (const rel of SYNCED_FILES) {
    it(`${rel} is byte-identical across ${SCENARIOS.join(", ")}`, () => {
      const contents = SCENARIOS.map((s) => {
        const abs = resolve(REPO_ROOT, "scenarios", s, rel);
        expect(existsSync(abs), `missing ${abs}`).toBe(true);
        return { scenario: s, abs, body: readFileSync(abs) };
      });

      const reference = contents[0]!;
      for (let i = 1; i < contents.length; i++) {
        const other = contents[i]!;
        const equal = reference.body.equals(other.body);
        // On mismatch, surface a unified-diff-friendly preview so the
        // failure tells the user exactly which scenario drifted.
        if (!equal) {
          console.error(
            `DRIFT: ${rel}\n  authoritative: ${reference.abs}\n  drifted:       ${other.abs}\n  fix: copy the authoritative file over the drifted one (or update SYNCED_FILES if the divergence is intentional).`,
          );
        }
        expect(equal, `${rel} differs between audio-tools and ${other.scenario}`).toBe(true);
      }
    });
  }
});
