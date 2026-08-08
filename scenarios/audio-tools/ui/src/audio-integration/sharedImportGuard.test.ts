import { describe, expect, it } from "vitest";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const AUDIO_ROOT = dirname(fileURLToPath(import.meta.url));
const removedProviderFile = ["Voice", "Stream", "Provider"].join("") + ".ts";
const UI_SOURCE = resolve(AUDIO_ROOT, "..");

function sourceFiles(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const stats = statSync(full);
    if (stats.isDirectory()) {
      if (entry === "node_modules" || entry === "dist") continue;
      sourceFiles(full, out);
    } else if (/\.(ts|tsx)$/.test(entry)) {
      out.push(full);
    }
  }
  return out;
}

describe("shared audio substrate boundary", () => {
  it("keeps orchestration in the shared browser package", () => {
    const hook = readFileSync(join(AUDIO_ROOT, "hooks", "useVoiceCore.ts"), "utf8");
    expect(hook).toContain('from "./useVoiceInput"');
    expect(hook).toContain("useAdoptedVoiceInput");
    expect(hook).not.toContain("const INITIAL_STATE");
    expect(hook).not.toContain("useState(");
  });

  it("does not retain a private PCM implementation", () => {
    const provider = readFileSync(
      join(AUDIO_ROOT, "hooks", "voice", "PcmVoiceStreamProvider.ts"),
      "utf8",
    );
    expect(provider).toContain('from "@vrooli/audio-capture-browser"');
    expect(provider).not.toMatch(/export class PcmVoiceStreamProvider/);
    expect(existsSync(resolve(AUDIO_ROOT, "hooks", "voice", removedProviderFile))).toBe(false);
  });

  it("rejects the deleted legacy provider symbol anywhere in UI source", () => {
    const offenders = sourceFiles(UI_SOURCE).filter((file) =>
      /\bVoiceStreamProvider\b/.test(readFileSync(file, "utf8"))
    );
    expect(offenders).toEqual([]);
  });
});
