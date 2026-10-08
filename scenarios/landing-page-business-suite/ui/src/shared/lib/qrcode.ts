/**
 * Minimal QR Code encoder (ISO/IEC 18004) for short UTF-8 payloads such as
 * otpauth:// enrollment URIs. Byte mode, error-correction level M, automatic
 * version (1-40) and mask selection.
 *
 * Structure follows Project Nayuki's reference QR Code generator (MIT).
 */

const ECC_CODEWORDS_PER_BLOCK_M = [-1, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28];
const NUM_ERROR_CORRECTION_BLOCKS_M = [-1, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49];
const ECC_FORMAT_BITS_M = 0;
const PENALTY_N1 = 3;
const PENALTY_N2 = 3;
const PENALTY_N3 = 40;
const PENALTY_N4 = 10;

/** Checked read: the tables and grids here are fully populated, so a miss is a bug. */
function at<T>(values: readonly T[], index: number): T {
  const value = values[index];
  if (value === undefined) throw new RangeError(`QR index ${String(index)} out of range`);
  return value;
}

export interface QrMatrix {
  size: number;
  /** modules[y][x] is true for a dark module. */
  modules: boolean[][];
}

export function encodeQr(text: string): QrMatrix {
  const data = Array.from(new TextEncoder().encode(text));
  let version = 1;
  for (; ; version++) {
    if (version > 40) throw new RangeError('QR payload too long');
    const countBits = version < 10 ? 8 : 16;
    const capacityBits = numDataCodewords(version) * 8;
    if (4 + countBits + data.length * 8 <= capacityBits) break;
  }

  const bits: number[] = [];
  appendBits(0x4, 4, bits);
  appendBits(data.length, version < 10 ? 8 : 16, bits);
  for (const byte of data) appendBits(byte, 8, bits);
  const capacityBits = numDataCodewords(version) * 8;
  appendBits(0, Math.min(4, capacityBits - bits.length), bits);
  appendBits(0, (8 - (bits.length % 8)) % 8, bits);
  for (let pad = 0xec; bits.length < capacityBits; pad ^= 0xec ^ 0x11) appendBits(pad, 8, bits);

  const codewords: number[] = [];
  for (let i = 0; i < bits.length; i += 8) {
    let value = 0;
    for (let j = 0; j < 8; j++) value = (value << 1) | at(bits, i + j);
    codewords.push(value);
  }

  const qr = new Builder(version);
  qr.drawFunctionPatterns();
  qr.drawCodewords(addEccAndInterleave(codewords, version));
  let bestMask = 0;
  let minPenalty = Infinity;
  for (let mask = 0; mask < 8; mask++) {
    qr.applyMask(mask);
    qr.drawFormatBits(mask);
    const penalty = qr.penaltyScore();
    if (penalty < minPenalty) {
      bestMask = mask;
      minPenalty = penalty;
    }
    qr.applyMask(mask);
  }
  qr.applyMask(bestMask);
  qr.drawFormatBits(bestMask);
  return { size: qr.size, modules: qr.modules };
}

function appendBits(value: number, length: number, out: number[]): void {
  for (let i = length - 1; i >= 0; i--) out.push((value >>> i) & 1);
}

function numRawDataModules(version: number): number {
  let result = (16 * version + 128) * version + 64;
  if (version >= 2) {
    const numAlign = Math.floor(version / 7) + 2;
    result -= (25 * numAlign - 10) * numAlign - 55;
    if (version >= 7) result -= 36;
  }
  return result;
}

function numDataCodewords(version: number): number {
  return Math.floor(numRawDataModules(version) / 8) - at(ECC_CODEWORDS_PER_BLOCK_M, version) * at(NUM_ERROR_CORRECTION_BLOCKS_M, version);
}

function addEccAndInterleave(data: number[], version: number): number[] {
  const numBlocks = at(NUM_ERROR_CORRECTION_BLOCKS_M, version);
  const blockEccLen = at(ECC_CODEWORDS_PER_BLOCK_M, version);
  const rawCodewords = Math.floor(numRawDataModules(version) / 8);
  const numShortBlocks = numBlocks - (rawCodewords % numBlocks);
  const shortBlockLen = Math.floor(rawCodewords / numBlocks);
  const divisor = reedSolomonDivisor(blockEccLen);
  const blocks: number[][] = [];
  for (let i = 0, k = 0; i < numBlocks; i++) {
    const dataLen = shortBlockLen - blockEccLen + (i < numShortBlocks ? 0 : 1);
    const block = data.slice(k, k + dataLen);
    k += dataLen;
    const ecc = reedSolomonRemainder(block, divisor);
    // Short blocks carry a placeholder so every block indexes its ECC alike;
    // interleaving skips that position for short blocks.
    if (i < numShortBlocks) block.push(0);
    blocks.push(block.concat(ecc));
  }
  const result: number[] = [];
  for (let i = 0; i < at(blocks, 0).length; i++) {
    blocks.forEach((block, j) => {
      if (i !== shortBlockLen - blockEccLen || j >= numShortBlocks) result.push(at(block, i));
    });
  }
  return result;
}

function reedSolomonDivisor(degree: number): number[] {
  const result = new Array<number>(degree - 1).fill(0).concat([1]);
  let root = 1;
  for (let i = 0; i < degree; i++) {
    for (let j = 0; j < result.length; j++) {
      result[j] = reedSolomonMultiply(at(result, j), root);
      if (j + 1 < result.length) result[j] = at(result, j) ^ at(result, j + 1);
    }
    root = reedSolomonMultiply(root, 0x02);
  }
  return result;
}

function reedSolomonRemainder(data: number[], divisor: number[]): number[] {
  const result = divisor.map(() => 0);
  for (const byte of data) {
    const factor = byte ^ (result.shift() ?? 0);
    result.push(0);
    divisor.forEach((coef, i) => { result[i] = at(result, i) ^ reedSolomonMultiply(coef, factor); });
  }
  return result;
}

function reedSolomonMultiply(x: number, y: number): number {
  let z = 0;
  for (let i = 7; i >= 0; i--) {
    z = (z << 1) ^ ((z >>> 7) * 0x11d);
    z ^= ((y >>> i) & 1) * x;
  }
  return z & 0xff;
}

class Builder {
  readonly size: number;
  readonly modules: boolean[][];
  private readonly isFunction: boolean[][];

  constructor(private readonly version: number) {
    this.size = version * 4 + 17;
    this.modules = Array.from({ length: this.size }, () => new Array<boolean>(this.size).fill(false));
    this.isFunction = Array.from({ length: this.size }, () => new Array<boolean>(this.size).fill(false));
  }

  drawFunctionPatterns(): void {
    for (let i = 0; i < this.size; i++) {
      this.setFunction(6, i, i % 2 === 0);
      this.setFunction(i, 6, i % 2 === 0);
    }
    this.drawFinder(3, 3);
    this.drawFinder(this.size - 4, 3);
    this.drawFinder(3, this.size - 4);
    const positions = this.alignmentPositions();
    const count = positions.length;
    for (let i = 0; i < count; i++) {
      for (let j = 0; j < count; j++) {
        if (!((i === 0 && j === 0) || (i === 0 && j === count - 1) || (i === count - 1 && j === 0))) {
          this.drawAlignment(at(positions, i), at(positions, j));
        }
      }
    }
    this.drawFormatBits(0);
    this.drawVersion();
  }

  drawFormatBits(mask: number): void {
    const data = (ECC_FORMAT_BITS_M << 3) | mask;
    let rem = data;
    for (let i = 0; i < 10; i++) rem = (rem << 1) ^ ((rem >>> 9) * 0x537);
    const bits = ((data << 10) | rem) ^ 0x5412;
    for (let i = 0; i <= 5; i++) this.setFunction(8, i, bit(bits, i));
    this.setFunction(8, 7, bit(bits, 6));
    this.setFunction(8, 8, bit(bits, 7));
    this.setFunction(7, 8, bit(bits, 8));
    for (let i = 9; i < 15; i++) this.setFunction(14 - i, 8, bit(bits, i));
    for (let i = 0; i < 8; i++) this.setFunction(this.size - 1 - i, 8, bit(bits, i));
    for (let i = 8; i < 15; i++) this.setFunction(8, this.size - 15 + i, bit(bits, i));
    this.setFunction(8, this.size - 8, true);
  }

  private drawVersion(): void {
    if (this.version < 7) return;
    let rem = this.version;
    for (let i = 0; i < 12; i++) rem = (rem << 1) ^ ((rem >>> 11) * 0x1f25);
    const bits = (this.version << 12) | rem;
    for (let i = 0; i < 18; i++) {
      const color = bit(bits, i);
      const a = this.size - 11 + (i % 3);
      const b = Math.floor(i / 3);
      this.setFunction(a, b, color);
      this.setFunction(b, a, color);
    }
  }

  private drawFinder(x: number, y: number): void {
    for (let dy = -4; dy <= 4; dy++) {
      for (let dx = -4; dx <= 4; dx++) {
        const dist = Math.max(Math.abs(dx), Math.abs(dy));
        const xx = x + dx;
        const yy = y + dy;
        if (xx >= 0 && xx < this.size && yy >= 0 && yy < this.size) this.setFunction(xx, yy, dist !== 2 && dist !== 4);
      }
    }
  }

  private drawAlignment(x: number, y: number): void {
    for (let dy = -2; dy <= 2; dy++) {
      for (let dx = -2; dx <= 2; dx++) this.setFunction(x + dx, y + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
    }
  }

  private alignmentPositions(): number[] {
    if (this.version === 1) return [];
    const numAlign = Math.floor(this.version / 7) + 2;
    const step = this.version === 32 ? 26 : Math.ceil((this.version * 4 + 4) / (numAlign * 2 - 2)) * 2;
    const result = [6];
    for (let pos = this.size - 7; result.length < numAlign; pos -= step) result.splice(1, 0, pos);
    return result;
  }

  private setFunction(x: number, y: number, dark: boolean): void {
    at(this.modules, y)[x] = dark;
    at(this.isFunction, y)[x] = true;
  }

  drawCodewords(data: number[]): void {
    let i = 0;
    for (let right = this.size - 1; right >= 1; right -= 2) {
      if (right === 6) right = 5;
      for (let vert = 0; vert < this.size; vert++) {
        for (let j = 0; j < 2; j++) {
          const x = right - j;
          const upward = ((right + 1) & 2) === 0;
          const y = upward ? this.size - 1 - vert : vert;
          if (!at(at(this.isFunction, y), x) && i < data.length * 8) {
            at(this.modules, y)[x] = bit(at(data, i >>> 3), 7 - (i & 7));
            i++;
          }
        }
      }
    }
  }

  applyMask(mask: number): void {
    for (let y = 0; y < this.size; y++) {
      for (let x = 0; x < this.size; x++) {
        let invert: boolean;
        switch (mask) {
          case 0: invert = (x + y) % 2 === 0; break;
          case 1: invert = y % 2 === 0; break;
          case 2: invert = x % 3 === 0; break;
          case 3: invert = (x + y) % 3 === 0; break;
          case 4: invert = (Math.floor(x / 3) + Math.floor(y / 2)) % 2 === 0; break;
          case 5: invert = ((x * y) % 2) + ((x * y) % 3) === 0; break;
          case 6: invert = (((x * y) % 2) + ((x * y) % 3)) % 2 === 0; break;
          default: invert = (((x + y) % 2) + ((x * y) % 3)) % 2 === 0; break;
        }
        if (!at(at(this.isFunction, y), x) && invert) at(this.modules, y)[x] = !at(at(this.modules, y), x);
      }
    }
  }

  penaltyScore(): number {
    let result = 0;
    const size = this.size;
    for (let y = 0; y < size; y++) {
      let runColor = false;
      let runX = 0;
      const history = [0, 0, 0, 0, 0, 0, 0];
      for (let x = 0; x < size; x++) {
        if (this.cell(x, y) === runColor) {
          runX++;
          if (runX === 5) result += PENALTY_N1;
          else if (runX > 5) result++;
        } else {
          this.addHistory(runX, history);
          if (!runColor) result += this.countFinderLike(history) * PENALTY_N3;
          runColor = this.cell(x, y);
          runX = 1;
        }
      }
      result += this.terminateAndCount(runColor, runX, history) * PENALTY_N3;
    }
    for (let x = 0; x < size; x++) {
      let runColor = false;
      let runY = 0;
      const history = [0, 0, 0, 0, 0, 0, 0];
      for (let y = 0; y < size; y++) {
        if (this.cell(x, y) === runColor) {
          runY++;
          if (runY === 5) result += PENALTY_N1;
          else if (runY > 5) result++;
        } else {
          this.addHistory(runY, history);
          if (!runColor) result += this.countFinderLike(history) * PENALTY_N3;
          runColor = this.cell(x, y);
          runY = 1;
        }
      }
      result += this.terminateAndCount(runColor, runY, history) * PENALTY_N3;
    }
    for (let y = 0; y < size - 1; y++) {
      for (let x = 0; x < size - 1; x++) {
        const color = this.cell(x, y);
        if (color === this.cell(x + 1, y) && color === this.cell(x, y + 1) && color === this.cell(x + 1, y + 1)) result += PENALTY_N2;
      }
    }
    let dark = 0;
    for (const row of this.modules) dark += row.filter(Boolean).length;
    const total = size * size;
    const k = Math.ceil(Math.abs(dark * 20 - total * 10) / total) - 1;
    result += k * PENALTY_N4;
    return result;
  }

  private cell(x: number, y: number): boolean {
    return at(at(this.modules, y), x);
  }

  private addHistory(run: number, history: number[]): void {
    if (history[0] === 0) run += this.size;
    history.pop();
    history.unshift(run);
  }

  private countFinderLike(history: number[]): number {
    const [h0, n, h2, h3, h4, h5, h6] = [0, 1, 2, 3, 4, 5, 6].map((index) => at(history, index)) as [number, number, number, number, number, number, number];
    const core = n > 0 && h2 === n && h3 === n * 3 && h4 === n && h5 === n;
    return (core && h0 >= n * 4 && h6 >= n ? 1 : 0) + (core && h6 >= n * 4 && h0 >= n ? 1 : 0);
  }

  private terminateAndCount(runColor: boolean, run: number, history: number[]): number {
    if (runColor) {
      this.addHistory(run, history);
      run = 0;
    }
    run += this.size;
    this.addHistory(run, history);
    return this.countFinderLike(history);
  }
}

function bit(value: number, index: number): boolean {
  return ((value >>> index) & 1) !== 0;
}
