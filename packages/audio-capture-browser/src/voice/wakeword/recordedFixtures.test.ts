import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { MfccDtwEngine } from "./engine";
import { MFCC_SAMPLE_RATE } from "./extractFromBytes";
import type { AudioFeatures } from "./types";

const fixtureRoot = join(dirname(fileURLToPath(import.meta.url)), "__fixtures__", "wakeword");
const haveFixtures = ["enroll", "same", "different"].every((name) => existsSync(join(fixtureRoot, name)));

function decodeWav16(bytes: Buffer): { pcm: Float32Array; sampleRate: number } {
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let sampleRate = MFCC_SAMPLE_RATE;
  let dataOffset = -1;
  let dataLength = 0;
  let offset = 12;
  while (offset + 8 <= view.byteLength) {
    const id = String.fromCharCode(view.getUint8(offset), view.getUint8(offset + 1), view.getUint8(offset + 2), view.getUint8(offset + 3));
    const size = view.getUint32(offset + 4, true);
    const body = offset + 8;
    if (id === "fmt ") sampleRate = view.getUint32(body + 4, true);
    if (id === "data") { dataOffset = body; dataLength = size; break; }
    offset = body + size + (size % 2);
  }
  if (dataOffset < 0) throw new Error("WAV has no data chunk");
  const pcm = new Float32Array(Math.floor(dataLength / 2));
  for (let i = 0; i < pcm.length; i++) pcm[i] = view.getInt16(dataOffset + i * 2, true) / 32_768;
  return { pcm, sampleRate };
}

function loadDirectory(engine: MfccDtwEngine, name: string): AudioFeatures[] {
  return readdirSync(join(fixtureRoot, name)).filter((file) => file.endsWith(".wav")).map((file) => {
    const decoded = decodeWav16(readFileSync(join(fixtureRoot, name, file)));
    return engine.extractFeatures(decoded.pcm, decoded.sampleRate);
  });
}

describe.skipIf(!haveFixtures)("recorded-fixture wake-word separation", () => {
  it("scores same-word recordings above different-word recordings", () => {
    const engine = new MfccDtwEngine();
    const enroll = loadDirectory(engine, "enroll");
    const same = loadDirectory(engine, "same");
    const different = loadDirectory(engine, "different");
    expect(enroll.length).toBeGreaterThanOrEqual(3);
    const calibration = engine.calibrate(enroll);
    expect(calibration).not.toBeNull();
    const sameScores = same.map((sample) => engine.compareBest(sample, enroll, 0.7, calibration).score).sort((a, b) => a - b);
    const differentScores = different.map((sample) => engine.compareBest(sample, enroll, 0.7, calibration).score);
    const sameMedian = sameScores[Math.floor(sameScores.length / 2)] ?? 0;
    const differentMax = Math.max(...differentScores);
    expect(sameMedian).toBeGreaterThanOrEqual(0.8);
    expect(differentMax).toBeLessThanOrEqual(0.4);
    expect(sameMedian - differentMax).toBeGreaterThanOrEqual(0.3);
  });
});

describe.runIf(!haveFixtures)("recorded-fixture wake-word separation (skipped)", () => {
  it("records that real fixtures are absent; synthetic tests remain the CI gate", () => {
    expect(haveFixtures).toBe(false);
  });
});
