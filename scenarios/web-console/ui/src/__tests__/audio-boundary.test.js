import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
/**
 * Audio adoption boundary guard.
 *
 * `src/audio-integration/` is web-console's audio surface. After the
 * "UI↔own-API only" migration, this folder calls web-console's own
 * AudioAdminService + AudioRuntimeService via the same-origin Connect
 * transport. It does NOT import from `@vrooli/proto-types/audio-tools/*`,
 * the retired foreign audio package, or any other foreign-scenario audio package.
 */
const THIS_FILE = fileURLToPath(import.meta.url);
const SRC_ROOT = resolve(THIS_FILE, "..", "..");
const AUDIO_SURFACE = join(SRC_ROOT, "audio-integration");
const MIC_OWNERSHIP_FILE = join(SRC_ROOT, "audio-integration", "hooks", "voice", "micOwnership.ts");
function walk(dir, out = []) {
    for (const entry of readdirSync(dir)) {
        const full = join(dir, entry);
        const st = statSync(full);
        if (st.isDirectory()) {
            if (entry === "node_modules" || entry === "dist")
                continue;
            walk(full, out);
            continue;
        }
        if (/\.(ts|tsx)$/.test(entry))
            out.push(full);
    }
    return out;
}
const IMPORT_RE = /\bfrom\s+["']([^"']+)["']/g;
describe("frontend audio adoption boundary", () => {
    it("the audio-integration folder exists with the expected entry points", () => {
        expect(existsSync(AUDIO_SURFACE)).toBe(true);
        expect(existsSync(join(AUDIO_SURFACE, "index.ts"))).toBe(true);
        expect(existsSync(join(AUDIO_SURFACE, "api"))).toBe(true);
        expect(existsSync(join(AUDIO_SURFACE, "hooks"))).toBe(true);
    });
    it("no UI file imports from foreign-scenario audio packages", () => {
        const files = walk(SRC_ROOT);
        const violations = [];
        for (const file of files) {
            const text = readFileSync(file, "utf8");
            IMPORT_RE.lastIndex = 0;
            let match;
            while ((match = IMPORT_RE.exec(text)) !== null) {
                const spec = match[1];
                if (typeof spec !== "string")
                    continue;
                if (spec === ["@audio-tools", "embed"].join("/") ||
                    spec.startsWith(["@audio-tools", "embed"].join("/") + "/") ||
                    spec.startsWith("@vrooli/proto-types/audio-tools/")) {
                    violations.push({ file, spec });
                }
            }
        }
        expect(violations, "UI must not import audio-tools proto types or the retired foreign audio package. " +
            "Use web-console's own audio_admin / audio_runtime proto types instead. " +
            "Offenders:\n" +
            violations.map((v) => `  ${v.file}\n    -> "${v.spec}"`).join("\n")).toEqual([]);
    });
    it("production mic acquisition goes through the ownership registry", () => {
        const files = walk(SRC_ROOT).filter((file) => {
            if (file === MIC_OWNERSHIP_FILE)
                return false;
            if (/\.test\.(ts|tsx)$/.test(file))
                return false;
            if (file.includes(`${join("src", "__tests__")}${"/"}`))
                return false;
            return true;
        });
        const violations = files.filter((file) => (readFileSync(file, "utf8").includes("navigator.mediaDevices.getUserMedia")));
        expect(violations, "Production code must acquire browser mic streams only through " +
            "audio-integration/hooks/voice/micOwnership.ts so leases are observable " +
            "and release through the registry. Offenders:\n" +
            violations.map((file) => `  ${file}`).join("\n")).toEqual([]);
    });
});
