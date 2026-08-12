// Recorded-fixture validation harness for wake-word scoring.
//
// This is the REAL-AUDIO counterpart to the synthetic separation test in
// engine.test.ts. It calibrates on a set of enrollment recordings and asserts
// that held-out same-word recordings score high while different-word / noise
// recordings score low — on actual speech, not synthetic tones.
//
// Fixtures are NOT committed (recording policy + size). The harness therefore
// auto-skips when the fixtures directory is absent (the synthetic harness is the
// CI gate). To run it locally:
//   1. Record mono 16-bit PCM WAV @ 16 kHz of your wake phrase and place them in:
//        __fixtures__/wakeword/enroll/   (>= 3 takes, your wake phrase)
//        __fixtures__/wakeword/same/     (>= 5 more takes, same phrase)
//        __fixtures__/wakeword/different/(>= 5 takes: other words / noise)
//      (relative to this directory).
//   2. `pnpm exec vitest run src/audio-integration/hooks/voice/wakeword/recordedFixtures.test.ts`
// Targets (mirror §9.2 of the plan): same-word median >= 0.8, different max <= 0.4,
// margin >= 0.3.
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { MfccDtwEngine } from "./engine";
import { MFCC_SAMPLE_RATE } from "./extractFromBytes";
const HERE = dirname(fileURLToPath(import.meta.url));
const FIXTURE_ROOT = join(HERE, "__fixtures__", "wakeword");
const HAVE_FIXTURES = existsSync(join(FIXTURE_ROOT, "enroll")) &&
    existsSync(join(FIXTURE_ROOT, "same")) &&
    existsSync(join(FIXTURE_ROOT, "different"));
/** Minimal mono 16-bit PCM WAV → Float32Array decoder (no Web Audio needed). */
function decodeWav16(bytes) {
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    // Walk RIFF chunks to find "fmt " and "data".
    let sampleRate = MFCC_SAMPLE_RATE;
    let dataOffset = -1;
    let dataLen = 0;
    let offset = 12; // skip "RIFF"<size>"WAVE"
    while (offset + 8 <= view.byteLength) {
        const id = String.fromCharCode(view.getUint8(offset), view.getUint8(offset + 1), view.getUint8(offset + 2), view.getUint8(offset + 3));
        const size = view.getUint32(offset + 4, true);
        const body = offset + 8;
        if (id === "fmt ")
            sampleRate = view.getUint32(body + 4, true);
        else if (id === "data") {
            dataOffset = body;
            dataLen = size;
            break;
        }
        offset = body + size + (size % 2);
    }
    if (dataOffset < 0)
        throw new Error("WAV has no data chunk");
    const count = Math.floor(dataLen / 2);
    const pcm = new Float32Array(count);
    for (let i = 0; i < count; i++) {
        pcm[i] = view.getInt16(dataOffset + i * 2, true) / 32768;
    }
    return { pcm, sampleRate };
}
function loadDir(engine, sub) {
    const dir = join(FIXTURE_ROOT, sub);
    return readdirSync(dir)
        .filter((f) => f.toLowerCase().endsWith(".wav"))
        .map((f) => {
        const { pcm, sampleRate } = decodeWav16(readFileSync(join(dir, f)));
        return engine.extractFeatures(pcm, sampleRate);
    });
}
describe.skipIf(!HAVE_FIXTURES)("recorded-fixture wake-word separation", () => {
    it("scores same-word high and different-word low on real audio", () => {
        const engine = new MfccDtwEngine();
        const enroll = loadDir(engine, "enroll");
        const same = loadDir(engine, "same");
        const different = loadDir(engine, "different");
        expect(enroll.length).toBeGreaterThanOrEqual(3);
        const cal = engine.calibrate(enroll);
        expect(cal).not.toBeNull();
        const sameScores = same.map((c) => engine.compareBest(c, enroll, 0.7, cal).score);
        const diffScores = different.map((c) => engine.compareBest(c, enroll, 0.7, cal).score);
        const median = (xs) => {
            const sorted = [...xs].sort((a, b) => a - b);
            return sorted[Math.floor(sorted.length / 2)] ?? 0;
        };
        const sameMedian = median(sameScores);
        const diffMax = Math.max(...diffScores);
        console.info(`[wakeword fixtures] same median=${sameMedian.toFixed(3)} different max=${diffMax.toFixed(3)}`);
        expect(sameMedian).toBeGreaterThanOrEqual(0.8);
        expect(diffMax).toBeLessThanOrEqual(0.4);
        expect(sameMedian - diffMax).toBeGreaterThanOrEqual(0.3);
    });
});
// Visible breadcrumb when fixtures are absent so a skipped suite is not mistaken
// for full coverage.
describe.runIf(!HAVE_FIXTURES)("recorded-fixture wake-word separation (skipped)", () => {
    it("is skipped — no fixtures present (synthetic harness in engine.test.ts is the CI gate)", () => {
        expect(HAVE_FIXTURES).toBe(false);
    });
});
