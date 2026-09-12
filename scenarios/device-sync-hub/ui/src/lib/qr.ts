/**
 * Dependency-free QR code generator (byte mode, error-correction level L).
 *
 * Pairing codes are short ASCII strings, so a single fixed approach covers
 * every case we emit: byte-mode encoding, the smallest version that fits, mask
 * pattern 0. We deliberately do NOT pull in an npm QR library (the scenario
 * forbids new deps); this is a compact, self-contained encoder.
 *
 * The output is a square boolean matrix (true = dark module). The React
 * component renders it as an SVG so it scales crisply at any size. The matrix
 * is best-effort: callers always also show the code as text, so a QR a reader
 * can't decode is a degraded affordance, never a broken flow.
 */

// Galois field (GF(256)) log/antilog tables for Reed-Solomon, generator 0x11d.
const GF_EXP = new Uint8Array(512);
const GF_LOG = new Uint8Array(256);
(() => {
  let x = 1;
  for (let i = 0; i < 255; i++) {
    GF_EXP[i] = x;
    GF_LOG[x] = i;
    x <<= 1;
    if (x & 0x100) x ^= 0x11d;
  }
  for (let i = 255; i < 512; i++) GF_EXP[i] = GF_EXP[i - 255] ?? 0;
})();

const gfExp = (i: number): number => GF_EXP[i] ?? 0;
const gfLog = (i: number): number => GF_LOG[i] ?? 0;

const gfMul = (a: number, b: number): number =>
  a === 0 || b === 0 ? 0 : gfExp(gfLog(a) + gfLog(b));

/** Build the Reed-Solomon generator polynomial of the given degree. */
function rsGeneratorPoly(degree: number): number[] {
  let poly = [1];
  for (let i = 0; i < degree; i++) {
    const next = new Array<number>(poly.length + 1).fill(0);
    for (let j = 0; j < poly.length; j++) {
      next[j] = (next[j] ?? 0) ^ gfMul(poly[j] ?? 0, gfExp(i));
      next[j + 1] = (next[j + 1] ?? 0) ^ (poly[j] ?? 0);
    }
    poly = next;
  }
  return poly;
}

function rsEncode(data: number[], ecLen: number): number[] {
  const gen = rsGeneratorPoly(ecLen);
  const res = new Array<number>(ecLen).fill(0);
  for (const byte of data) {
    const factor = byte ^ (res[0] ?? 0);
    res.shift();
    res.push(0);
    for (let i = 0; i < gen.length; i++) {
      res[i] = (res[i] ?? 0) ^ gfMul(gen[i] ?? 0, factor);
    }
  }
  return res;
}

interface VersionSpec {
  capacity: number;
  ec: number;
  align: number[];
}

// Per-version (1..5), level L: data-codeword capacity + EC codewords + alignment
// pattern centres. Single-block versions only (sufficient for pairing codes).
const VERSION_TABLE: Record<number, VersionSpec> = {
  1: { capacity: 19, ec: 7, align: [] },
  2: { capacity: 34, ec: 10, align: [6, 18] },
  3: { capacity: 55, ec: 15, align: [6, 22] },
  4: { capacity: 80, ec: 20, align: [6, 26] },
  5: { capacity: 108, ec: 26, align: [6, 30] },
};

function pickVersion(byteLen: number): { version: number; spec: VersionSpec } {
  for (const v of [1, 2, 3, 4, 5]) {
    const spec = VERSION_TABLE[v];
    if (!spec) continue;
    // byte-mode header = 4-bit mode + 8-bit length (versions 1..9).
    const dataBits = 4 + 8 + byteLen * 8;
    if (Math.ceil(dataBits / 8) <= spec.capacity) return { version: v, spec };
  }
  throw new Error("QR payload too large for supported versions (<= v5)");
}

/** Encode `text` into a QR boolean matrix. ASCII / byte mode only, mask 0. */
export function encodeQr(text: string): boolean[][] {
  const bytes = Array.from(new TextEncoder().encode(text));
  const { version, spec } = pickVersion(bytes.length);
  const size = 17 + version * 4;

  // --- Bit stream: mode (0100) + length (8 bits) + data + terminator + pad. ---
  const bits: number[] = [];
  const pushBits = (value: number, len: number) => {
    for (let i = len - 1; i >= 0; i--) bits.push((value >> i) & 1);
  };
  pushBits(0b0100, 4);
  pushBits(bytes.length, 8);
  for (const b of bytes) pushBits(b, 8);
  const capacityBits = spec.capacity * 8;
  for (let i = 0; i < 4 && bits.length < capacityBits; i++) bits.push(0);
  while (bits.length % 8 !== 0) bits.push(0);
  const dataCodewords: number[] = [];
  for (let i = 0; i < bits.length; i += 8) {
    let byte = 0;
    for (let j = 0; j < 8; j++) byte = (byte << 1) | (bits[i + j] ?? 0);
    dataCodewords.push(byte);
  }
  const padBytes = [0xec, 0x11];
  let p = 0;
  while (dataCodewords.length < spec.capacity) {
    dataCodewords.push(padBytes[p % 2] ?? 0);
    p++;
  }
  const allCodewords = [...dataCodewords, ...rsEncode(dataCodewords, spec.ec)];

  // --- Matrix scaffolding (flat Uint8Array buffers; 0=light, 1=dark). ---
  const modules = new Uint8Array(size * size);
  const reserved = new Uint8Array(size * size);
  const at = (r: number, c: number) => r * size + c;
  const set = (r: number, c: number, dark: boolean, reserve = true) => {
    modules[at(r, c)] = dark ? 1 : 0;
    if (reserve) reserved[at(r, c)] = 1;
  };
  const isReserved = (r: number, c: number) => reserved[at(r, c)] === 1;

  const placeFinder = (r0: number, c0: number) => {
    for (let r = -1; r <= 7; r++) {
      for (let c = -1; c <= 7; c++) {
        const rr = r0 + r;
        const cc = c0 + c;
        if (rr < 0 || rr >= size || cc < 0 || cc >= size) continue;
        const inRing =
          (r >= 0 && r <= 6 && (c === 0 || c === 6)) ||
          (c >= 0 && c <= 6 && (r === 0 || r === 6));
        const inCore = r >= 2 && r <= 4 && c >= 2 && c <= 4;
        set(rr, cc, inRing || inCore);
      }
    }
  };
  placeFinder(0, 0);
  placeFinder(0, size - 7);
  placeFinder(size - 7, 0);

  // Timing patterns.
  for (let i = 8; i < size - 8; i++) {
    set(6, i, i % 2 === 0);
    set(i, 6, i % 2 === 0);
  }
  // Dark module.
  set(size - 8, 8, true);

  // Alignment patterns (skip those overlapping a finder).
  for (const ar of spec.align) {
    for (const ac of spec.align) {
      if ((ar === 6 && ac === 6) || (ar === 6 && ac === size - 7) || (ar === size - 7 && ac === 6)) {
        continue;
      }
      for (let r = -2; r <= 2; r++) {
        for (let c = -2; c <= 2; c++) {
          set(ar + r, ac + c, Math.max(Math.abs(r), Math.abs(c)) !== 1);
        }
      }
    }
  }

  // Reserve format-info areas (filled below).
  for (let i = 0; i < 9; i++) {
    reserved[at(8, i)] = 1;
    reserved[at(i, 8)] = 1;
  }
  for (let i = 0; i < 8; i++) {
    reserved[at(8, size - 1 - i)] = 1;
    reserved[at(size - 1 - i, 8)] = 1;
  }

  // --- Place data bits in zig-zag, applying mask pattern 0 (r+c even). ---
  let bitIdx = 0;
  const totalBits = allCodewords.length * 8;
  let col = size - 1;
  let upward = true;
  while (col > 0) {
    if (col === 6) col--; // skip the vertical timing column
    for (let n = 0; n < size; n++) {
      const row = upward ? size - 1 - n : n;
      for (let dc = 0; dc < 2; dc++) {
        const c = col - dc;
        if (isReserved(row, c)) continue;
        let dark = false;
        if (bitIdx < totalBits) {
          const cw = allCodewords[bitIdx >> 3] ?? 0;
          dark = ((cw >> (7 - (bitIdx % 8))) & 1) === 1;
          bitIdx++;
        }
        if ((row + c) % 2 === 0) dark = !dark; // mask 0
        modules[at(row, c)] = dark ? 1 : 0;
      }
    }
    col -= 2;
    upward = !upward;
  }

  // --- Format information (EC level L = 0b01, mask 0 = 0b000). ---
  const formatBits = computeFormatBits(0b01, 0);
  const fmt = (i: number) => (formatBits[i] ?? 0) === 1;
  for (let i = 0; i <= 5; i++) modules[at(8, i)] = fmt(i) ? 1 : 0;
  modules[at(8, 7)] = fmt(6) ? 1 : 0;
  modules[at(8, 8)] = fmt(7) ? 1 : 0;
  modules[at(7, 8)] = fmt(8) ? 1 : 0;
  for (let i = 9; i < 15; i++) modules[at(14 - i, 8)] = fmt(i) ? 1 : 0;
  for (let i = 0; i < 8; i++) modules[at(8, size - 1 - i)] = fmt(i) ? 1 : 0;
  for (let i = 8; i < 15; i++) modules[at(size - 15 + i, 8)] = fmt(i) ? 1 : 0;

  // Materialise the flat buffer as a boolean matrix for the renderer.
  const out: boolean[][] = [];
  for (let r = 0; r < size; r++) {
    const row: boolean[] = [];
    for (let c = 0; c < size; c++) row.push(modules[at(r, c)] === 1);
    out.push(row);
  }
  return out;
}

/** 15-bit BCH-coded format information for (ecLevel, mask). */
function computeFormatBits(ecLevel: number, mask: number): number[] {
  const data = (ecLevel << 3) | mask;
  let rem = data;
  for (let i = 0; i < 10; i++) {
    rem = (rem << 1) ^ ((rem >> 9) & 1 ? 0b10100110111 : 0);
  }
  const combined = ((data << 10) | (rem & 0x3ff)) ^ 0b101010000010010;
  const out: number[] = [];
  for (let i = 14; i >= 0; i--) out.push((combined >> i) & 1);
  return out;
}
