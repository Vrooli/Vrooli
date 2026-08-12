import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
/**
 * Static-assertion tests enforcing the greenfield constraints of the
 * terminal-session refactor (see
 * §13). If any of these patterns reappear in the codebase, the
 * implementation has drifted back into legacy/compat territory and
 * the test fails loudly.
 */
// Resolve the src/ root from this test file's URL. Avoids reliance on
// the CommonJS __dirname global (absent under ESM + Vite).
const THIS_FILE = fileURLToPath(import.meta.url);
const SRC_ROOT = resolve(THIS_FILE, "..", "..");
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
const ALL_SOURCE_FILES = walk(SRC_ROOT);
const PROD_FILES = ALL_SOURCE_FILES.filter((p) => !/\/__tests__\//.test(p) && !/\.test\.(ts|tsx)$/.test(p));
function readIf(path) {
    try {
        return readFileSync(path, "utf8");
    }
    catch {
        return "";
    }
}
describe("greenfield: terminal-session rework", () => {
    it("does not reintroduce useTerminalSocket", () => {
        const offenders = ALL_SOURCE_FILES.filter((p) => !p.endsWith("greenfield-assertions.test.ts") &&
            readIf(p).includes("useTerminalSocket"));
        expect(offenders).toEqual([]);
    });
    it("does not strip DEC mode 2026 on the client (server is authoritative)", () => {
        // The server-side strip lives in api/ansi_responder.go. Any UI
        // code that also strips is belt-and-suspenders duplication.
        const pattern = /\\x1b\\\[\\\?2026/;
        const offenders = PROD_FILES.filter((p) => pattern.test(readIf(p)));
        expect(offenders).toEqual([]);
    });
    it("does not write \\b \\b erase sequences from local echo reconciliation", () => {
        // The plan explicitly removes \b \b erasure; xterm repaints from
        // the server-authoritative stream instead.
        const content = readIf(join(SRC_ROOT, "lib", "localEcho.ts"));
        expect(content).not.toMatch(/"\\b \\b"/);
    });
    it("defines sendInput / trySendStdin nowhere (submitInput via gate is the single path)", () => {
        // Test-helper handle shapes and mocks are allowed; production code
        // must never define these function names.
        const pattern = /\b(sendInput|trySendStdin)\s*[:=]\s*\(/;
        const offenders = PROD_FILES.filter((p) => pattern.test(readIf(p)));
        expect(offenders).toEqual([]);
    });
    it("does not reintroduce deleted history-cache symbols", () => {
        // Plan §3 / §10.3: greenfield. The byte-offset history protocol is
        // gone. None of these symbols may reappear in production code.
        const forbidden = [
            "history_offset",
            "outputHistory",
            "appendHistory",
            "OfflineBufferMax",
            "WC_OFFLINE_BUFFER_MAX",
            "totalBytesRef",
            "hasCachedState",
            "terminalCache",
            "loadTerminalCache",
            "saveTerminalCache",
            "MsgTypePTYState",
            "InitialAltBuffer",
        ];
        for (const sym of forbidden) {
            const offenders = PROD_FILES.filter((p) => readIf(p).includes(sym));
            expect(offenders, `${sym} reappeared in production: ${offenders.join(", ")}`).toEqual([]);
        }
    });
    it("every input source tag at gate.submit / submitInput call sites is a known InputSource", () => {
        // Type-level enforcement covers most of this, but scan for raw
        // string literals passed as the `source` argument. The regex
        // matches `.submit(<any>, "tag")` and `submitInput(<any>, "tag")`.
        const known = [
            "xterm",
            "toolbar-key",
            "toolbar-submit",
            "paste",
            "voice",
            "upload",
        ];
        // Only match source arguments that are simple identifier-like
        // strings to avoid false positives on payload literals that
        // happen to contain commas (e.g. path + "\n").
        const re = /\.(?:submit|submitInput)\([^,]+,\s*"([a-z][a-z-]*)"/g;
        for (const p of PROD_FILES) {
            const src = readIf(p);
            let m;
            while ((m = re.exec(src)) !== null) {
                const tag = m[1];
                expect(known).toContain(tag);
            }
        }
    });
});
