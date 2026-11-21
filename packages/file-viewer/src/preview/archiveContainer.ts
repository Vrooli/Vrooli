// Check the central directory before a parser inflates untrusted Office bytes.
export function checkArchiveContainer(bytes: ArrayBuffer, maxBytes: number, maxExpanded = 128 * 1024 * 1024): void {
  if (bytes.byteLength > maxBytes) throw new RangeError("bytes");
  const view = new DataView(bytes);
  if (bytes.byteLength < 4 || view.getUint32(0, true) !== 0x04034b50) return;
  let eocd = -1;
  for (let i = bytes.byteLength - 22; i >= Math.max(0, bytes.byteLength - 65557); i--) {
    if (view.getUint32(i, true) === 0x06054b50 && i + 22 + view.getUint16(i + 20, true) === bytes.byteLength) { eocd = i; break; }
  }
  if (eocd < 0) throw new TypeError("zip");
  const entries = view.getUint16(eocd + 10, true), start = view.getUint32(eocd + 16, true), size = view.getUint32(eocd + 12, true);
  if (entries === 65535 || entries > 10000 || start + size > eocd) throw new RangeError("zip limit");
  let offset = start, expanded = 0;
  for (let i = 0; i < entries; i++) {
    if (offset + 46 > start + size || view.getUint32(offset, true) !== 0x02014b50) throw new TypeError("zip directory");
    const uncompressed = view.getUint32(offset + 24, true);
    expanded += uncompressed;
    if (uncompressed > 64 * 1024 * 1024 || expanded > maxExpanded) throw new RangeError("zip expansion");
    offset += 46 + view.getUint16(offset + 28, true) + view.getUint16(offset + 30, true) + view.getUint16(offset + 32, true);
  }
  if (offset !== start + size) throw new TypeError("zip directory size");
}
