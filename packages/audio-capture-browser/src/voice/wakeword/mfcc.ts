// DOC: docs/internal/SEAMS.md#wake-word-engine-seam
//
// Pure-JS MFCC (Mel-Frequency Cepstral Coefficients) extraction.
// No npm dependencies — uses a hand-rolled radix-2 Cooley-Tukey FFT.
//
// Pipeline: pre-emphasis -> framing -> Hamming window -> FFT ->
//           mel filterbank -> log -> DCT -> 13 MFCCs per frame.
//
// Performance: <20ms for 2 seconds of 16kHz audio (~160 frames).

import {
  NUM_MFCC_COEFFICIENTS,
  FRAME_LENGTH_MS,
  FRAME_HOP_MS,
  MEL_FILTER_COUNT,
  MEL_LOW_HZ,
  MEL_HIGH_HZ,
} from "./types";

// ---------------------------------------------------------------------------
// FFT — radix-2 in-place Cooley-Tukey (iterative, no recursion)
// ---------------------------------------------------------------------------

/** Next power of 2 >= n. */
export function nextPow2(n: number): number {
  let p = 1;
  while (p < n) p <<= 1;
  return p;
}

/**
 * In-place radix-2 FFT. `real` and `imag` must have length = power of 2.
 * After return they contain the complex DFT coefficients.
 */
export function fft(real: Float64Array, imag: Float64Array): void {
  const n = real.length;
  // Bit-reversal permutation
  for (let i = 1, j = 0; i < n; i++) {
    let bit = n >> 1;
    while (j & bit) {
      j ^= bit;
      bit >>= 1;
    }
    j ^= bit;
    if (i < j) {
      const tr = real[i] ?? 0;
      real[i] = real[j] ?? 0;
      real[j] = tr;
      const ti = imag[i] ?? 0;
      imag[i] = imag[j] ?? 0;
      imag[j] = ti;
    }
  }
  // Butterfly passes
  for (let len = 2; len <= n; len <<= 1) {
    const halfLen = len >> 1;
    const angle = (-2 * Math.PI) / len;
    const wReal = Math.cos(angle);
    const wImag = Math.sin(angle);
    for (let i = 0; i < n; i += len) {
      let curReal = 1;
      let curImag = 0;
      for (let j = 0; j < halfLen; j++) {
        const idxHi = i + j + halfLen;
        const idxLo = i + j;
        const realHi = real[idxHi] ?? 0;
        const imagHi = imag[idxHi] ?? 0;
        const realLo = real[idxLo] ?? 0;
        const imagLo = imag[idxLo] ?? 0;
        const tReal = curReal * realHi - curImag * imagHi;
        const tImag = curReal * imagHi + curImag * realHi;
        real[idxHi] = realLo - tReal;
        imag[idxHi] = imagLo - tImag;
        real[idxLo] = realLo + tReal;
        imag[idxLo] = imagLo + tImag;
        const nextReal = curReal * wReal - curImag * wImag;
        curImag = curReal * wImag + curImag * wReal;
        curReal = nextReal;
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Mel filterbank
// ---------------------------------------------------------------------------

/** Convert frequency in Hz to mel scale. */
export function hzToMel(hz: number): number {
  return 2595 * Math.log10(1 + hz / 700);
}

/** Convert mel back to Hz. */
export function melToHz(mel: number): number {
  return 700 * (10 ** (mel / 2595) - 1);
}

/**
 * Build a mel filterbank as a 2D array: [numFilters][fftBins].
 * Each row is a triangular filter in the frequency domain.
 */
export function buildMelFilterbank(
  numFilters: number,
  fftSize: number,
  sampleRate: number,
  lowHz: number,
  highHz: number,
): Float64Array[] {
  const lowMel = hzToMel(lowHz);
  const highMel = hzToMel(highHz);
  const numBins = (fftSize >> 1) + 1;

  // Equally spaced mel points (numFilters + 2 for edges)
  const melPoints = new Float64Array(numFilters + 2);
  for (let i = 0; i < melPoints.length; i++) {
    melPoints[i] = lowMel + (i * (highMel - lowMel)) / (numFilters + 1);
  }

  // Convert to FFT bin indices
  const binIndices = new Float64Array(melPoints.length);
  for (let i = 0; i < melPoints.length; i++) {
    binIndices[i] = Math.floor(((fftSize + 1) * melToHz(melPoints[i] ?? 0)) / sampleRate);
  }

  const filters: Float64Array[] = [];
  for (let m = 0; m < numFilters; m++) {
    const filter = new Float64Array(numBins);
    const left = binIndices[m] ?? 0;
    const center = binIndices[m + 1] ?? 0;
    const right = binIndices[m + 2] ?? 0;

    for (let k = 0; k < numBins; k++) {
      if (k >= left && k <= center && center > left) {
        filter[k] = (k - left) / (center - left);
      } else if (k > center && k <= right && right > center) {
        filter[k] = (right - k) / (right - center);
      }
    }
    filters.push(filter);
  }
  return filters;
}

// ---------------------------------------------------------------------------
// DCT-II (type 2, orthogonal)
// ---------------------------------------------------------------------------

/**
 * Compute the first `numCoeffs` DCT-II coefficients of `input`.
 */
export function dctII(input: Float64Array, numCoeffs: number): Float64Array {
  const n = input.length;
  const output = new Float64Array(numCoeffs);
  const scale = Math.sqrt(2 / n);
  for (let k = 0; k < numCoeffs; k++) {
    let sum = 0;
    for (let i = 0; i < n; i++) {
      sum += (input[i] ?? 0) * Math.cos((Math.PI * k * (2 * i + 1)) / (2 * n));
    }
    output[k] = sum * scale;
  }
  return output;
}

// ---------------------------------------------------------------------------
// Hamming window (cached per frame size)
// ---------------------------------------------------------------------------

const hammingCache = new Map<number, Float64Array>();

function getHammingWindow(size: number): Float64Array {
  let win = hammingCache.get(size);
  if (!win) {
    win = new Float64Array(size);
    for (let i = 0; i < size; i++) {
      win[i] = 0.54 - 0.46 * Math.cos((2 * Math.PI * i) / (size - 1));
    }
    hammingCache.set(size, win);
  }
  return win;
}

// ---------------------------------------------------------------------------
// MFCC extraction — main entry point
// ---------------------------------------------------------------------------

/** Cached filterbank keyed by `${numFilters}-${fftSize}-${sampleRate}`. */
const filterbankCache = new Map<string, Float64Array[]>();

/**
 * Extract MFCC features from mono PCM Float32 audio.
 *
 * @param audio - Mono PCM samples in [-1, 1] range.
 * @param sampleRate - Sample rate of the input audio (e.g., 16000).
 * @returns 2D array of shape [numFrames][NUM_MFCC_COEFFICIENTS].
 */
export function extractMfcc(audio: Float32Array, sampleRate: number): number[][] {
  if (audio.length === 0) return [];

  const frameLenSamples = Math.round((FRAME_LENGTH_MS / 1000) * sampleRate);
  const frameHopSamples = Math.round((FRAME_HOP_MS / 1000) * sampleRate);
  const fftSize = nextPow2(frameLenSamples);
  const numBins = (fftSize >> 1) + 1;

  // Pre-emphasis (coefficient 0.97)
  const emphasized = new Float32Array(audio.length);
  emphasized[0] = audio[0] ?? 0;
  for (let i = 1; i < audio.length; i++) {
    emphasized[i] = (audio[i] ?? 0) - 0.97 * (audio[i - 1] ?? 0);
  }

  // Get or build mel filterbank
  const fbKey = `${MEL_FILTER_COUNT}-${fftSize}-${sampleRate}`;
  let filterbank = filterbankCache.get(fbKey);
  if (!filterbank) {
    filterbank = buildMelFilterbank(MEL_FILTER_COUNT, fftSize, sampleRate, MEL_LOW_HZ, MEL_HIGH_HZ);
    filterbankCache.set(fbKey, filterbank);
  }

  const hammingWin = getHammingWindow(frameLenSamples);
  const numFrames = Math.max(0, Math.floor((audio.length - frameLenSamples) / frameHopSamples) + 1);
  const mfccs: number[][] = [];

  // Reusable buffers
  const real = new Float64Array(fftSize);
  const imag = new Float64Array(fftSize);
  const powerSpectrum = new Float64Array(numBins);
  const melEnergies = new Float64Array(MEL_FILTER_COUNT);

  for (let f = 0; f < numFrames; f++) {
    const offset = f * frameHopSamples;

    // Window + zero-pad into FFT buffer
    real.fill(0);
    imag.fill(0);
    for (let i = 0; i < frameLenSamples; i++) {
      real[i] = (emphasized[offset + i] ?? 0) * (hammingWin[i] ?? 0);
    }

    // FFT
    fft(real, imag);

    // Power spectrum (|X(k)|^2 / N)
    for (let k = 0; k < numBins; k++) {
      const re = real[k] ?? 0;
      const im = imag[k] ?? 0;
      powerSpectrum[k] = (re * re + im * im) / fftSize;
    }

    // Apply mel filterbank
    for (let m = 0; m < MEL_FILTER_COUNT; m++) {
      let energy = 0;
      const filter = filterbank[m];
      if (!filter) continue;
      for (let k = 0; k < numBins; k++) {
        energy += (powerSpectrum[k] ?? 0) * (filter[k] ?? 0);
      }
      // Log with floor to avoid log(0)
      melEnergies[m] = Math.log(Math.max(energy, 1e-22));
    }

    // DCT to get MFCCs
    const coeffs = dctII(melEnergies, NUM_MFCC_COEFFICIENTS);
    mfccs.push(Array.from(coeffs));
  }

  return mfccs;
}
