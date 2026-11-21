import { deflateSync } from 'node:zlib';
import { isBlankPixels, measurePng } from '../../../src/capture/paint';

/** Encodes an RGB PNG with filter 0 so tests do not need an image library. */
export function rgbPng(
  width: number,
  height: number,
  pixel: (x: number, y: number) => number[]
): Buffer {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++)
    for (let x = 0; x < width; x++) raw.set(pixel(x, y), y * (width * 3 + 1) + 1 + x * 3);
  const chunk = (type: string, data: Buffer): Buffer => {
    const length = Buffer.alloc(4);
    length.writeUInt32BE(data.length);
    return Buffer.concat([length, Buffer.from(type, 'latin1'), data, Buffer.alloc(4)]);
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header.set([8, 2, 0, 0, 0], 8);
  return Buffer.concat([
    Buffer.from('89504e470d0a1a0a', 'hex'),
    chunk('IHDR', header),
    chunk('IDAT', deflateSync(raw)),
    chunk('IEND', Buffer.alloc(0)),
  ]);
}

describe('L0 blank-pixel detection', () => {
  it('reports a page-background-only screenshot as blank', () => {
    const stats = measurePng(rgbPng(390, 844, () => [249, 248, 244]));
    expect(stats).toMatchObject({ width: 390, height: 844 });
    expect(isBlankPixels(stats)).toBe(true);
  });

  it('keeps a sparse page with one short line of text', () => {
    // A 120×14 px "text line" is 0.5% of a 390×844 phone viewport.
    const stats = measurePng(
      rgbPng(390, 844, (x, y) =>
        x >= 16 && x < 136 && y >= 40 && y < 54 ? [30, 30, 30] : [255, 255, 255]
      )
    );
    expect(isBlankPixels(stats)).toBe(false);
  });

  it('rejects data that is not a PNG', () => {
    expect(() => measurePng(Buffer.from('viewport-pixels'))).toThrow('not a PNG image');
  });
});
