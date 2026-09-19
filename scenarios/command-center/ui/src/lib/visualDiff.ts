export interface PixelDiff { differentPixels: number; maxDelta: number; contrastWorstCase: number; }

/** Deterministic comparator used by the visual safety net and preview parity tests. */
export function comparePixels(expected: Uint8ClampedArray, actual: Uint8ClampedArray): PixelDiff {
  if (expected.length !== actual.length) return { differentPixels: Math.max(expected.length, actual.length), maxDelta: 255, contrastWorstCase: 0 };
  let differentPixels = 0; let maxDelta = 0; let contrastWorstCase = 255;
  const at = (values: Uint8ClampedArray, index: number) => values[index] ?? 0;
  for (let i = 0; i < expected.length; i += 4) {
    const delta = Math.max(Math.abs(at(expected, i) - at(actual, i)), Math.abs(at(expected, i + 1) - at(actual, i + 1)), Math.abs(at(expected, i + 2) - at(actual, i + 2)));
    if (delta > 0) differentPixels++;
    maxDelta = Math.max(maxDelta, delta);
    const expectedLuma = 0.2126 * at(expected, i) + 0.7152 * at(expected, i + 1) + 0.0722 * at(expected, i + 2);
    const actualLuma = 0.2126 * at(actual, i) + 0.7152 * at(actual, i + 1) + 0.0722 * at(actual, i + 2);
    contrastWorstCase = Math.min(contrastWorstCase, Math.abs(expectedLuma - actualLuma));
  }
  return { differentPixels, maxDelta, contrastWorstCase };
}

export const VISUAL_DIFF_PIN = { seed: "command-center-milestone-1", timestampsMs: [0, 250, 1000, 2500, 5000, 10000, 14000] } as const;

export interface DeterministicClock { now: () => number; advance: (timestampMs: number) => void; }
export function createDeterministicClock(initial = 0): DeterministicClock {
  let timestampMs = initial;
  return { now: () => timestampMs, advance: (next) => { timestampMs = next; } };
}
