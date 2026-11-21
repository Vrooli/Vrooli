// Bound raster decode area as well as ZIP bytes. Highly compressed images can
// otherwise exhaust memory even inside a small, structurally valid package.
export function rasterPixels(bytes: Uint8Array, name: string, maxEdge = 16384, maxPixels = 24000000): number {
  if (!/\.(png|jpe?g|gif|webp)$/i.test(name)) return 0;
  const v = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let width = 0, height = 0;
  if (bytes.length >= 24 && v.getUint32(0) === 0x89504e47 && v.getUint32(4) === 0x0d0a1a0a) { width = v.getUint32(16); height = v.getUint32(20); }
  else if (bytes.length >= 10 && String.fromCharCode(...bytes.subarray(0, 3)) === "GIF") { width = v.getUint16(6, true); height = v.getUint16(8, true); }
  else if (bytes.length >= 16 && v.getUint32(0) === 0x52494646 && v.getUint32(8) === 0x57454250) {
    const kind = v.getUint32(12);
    if (kind === 0x56503858 && bytes.length >= 30) { width = 1 + (v.getUint32(24, true) & 0xffffff); height = 1 + ((v.getUint32(26, true) >>> 8) & 0xffffff); }
    else if (kind === 0x5650384c && bytes.length >= 25) { const bits = v.getUint32(21, true); width = 1 + (bits & 0x3fff); height = 1 + ((bits >>> 14) & 0x3fff); }
    else if (kind === 0x56503820 && bytes.length >= 30) { width = v.getUint16(26, true) & 0x3fff; height = v.getUint16(28, true) & 0x3fff; }
  } else if (bytes.length >= 4 && v.getUint16(0) === 0xffd8) {
    let offset = 2;
    while (offset + 4 <= bytes.length) {
      if (bytes[offset] !== 0xff) break;
      const marker = bytes[offset + 1] ?? 0;
      if (marker === 0xff) { offset++; continue; }
      if (marker === 0xd9 || marker === 0xda) break;
      const length = v.getUint16(offset + 2);
      if (length < 2 || offset + 2 + length > bytes.length) break;
      if ([0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf].includes(marker) && length >= 7) { height = v.getUint16(offset + 5); width = v.getUint16(offset + 7); break; }
      offset += 2 + length;
    }
  }
  if (!width || !height) throw Error("invalid");
  if (width > maxEdge || height > maxEdge || width * height > maxPixels) throw new RangeError("pixels");
  return width * height;
}
