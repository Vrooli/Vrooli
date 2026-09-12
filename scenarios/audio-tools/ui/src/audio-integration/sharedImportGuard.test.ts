import { describe, expect, it } from "vitest";
import { existsSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

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

function unmarkedProductionFiles(dir: string): string[] {
  return sourceFiles(dir).filter((file) => {
    if (/\.test\.(ts|tsx)$/.test(file)) return false;
    const source = readFileSync(file, "utf8");
    return !source.includes("HOST DIFFERENCE:") && !isPurePackageReexport(source);
  });
}

function isPurePackageReexport(source: string): boolean {
  const withoutComments = source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/^\s*\/\/.*$/gm, "")
    .trim();
  return /^(?:export\s+(?:type\s+)?(?:\*|\{[\s\S]*?\})\s+from\s+["']@vrooli\/audio-capture-browser["'];?\s*)+$/.test(withoutComments);
}

describe("shared audio substrate boundary", () => {
  it("keeps orchestration in the shared browser package", () => {
    const hook = readFileSync(join(AUDIO_ROOT, "hooks", "useVoiceCore.ts"), "utf8");
    expect(hook).toContain('from "@vrooli/react-component-library/useVoiceInput/3"');
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

  it("rejects unmarked production forks in every consumer integration directory", () => {
    const repo = resolve(AUDIO_ROOT, "../../../../../");
    for (const scenario of ["audio-tools", "web-console", "swarm-manager"]) {
      expect(unmarkedProductionFiles(join(repo, "scenarios", scenario, "ui", "src", "audio-integration"))).toEqual([]);
    }
  });

  it("reports both remedies for an unmarked fork", () => {
    const fixtureRoot = mkdtempSync(join(tmpdir(), "audio-integration-guard-"));
    try {
      const fork = join(fixtureRoot, "fork.ts");
      writeFileSync(fork, "export const divergent = true;\n");
      expect(unmarkedProductionFiles(fixtureRoot)).toEqual([fork]);
      expect("import from @vrooli/audio-capture-browser or add HOST DIFFERENCE:").toContain("HOST DIFFERENCE:");
    } finally {
      rmSync(fixtureRoot, { recursive: true, force: true });
    }
  });

  it("rejects an implementation hidden behind a package import", () => {
    const fixtureRoot = mkdtempSync(join(tmpdir(), "audio-integration-import-fork-"));
    try {
      const fork = join(fixtureRoot, "fork.ts");
      writeFileSync(fork, 'import { downsample } from "@vrooli/audio-capture-browser";\nexport const divergent = downsample;\n');
      expect(unmarkedProductionFiles(fixtureRoot)).toEqual([fork]);
    } finally {
      rmSync(fixtureRoot, { recursive: true, force: true });
    }
  });
});
