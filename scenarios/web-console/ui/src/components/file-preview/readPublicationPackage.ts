import { rasterPixels } from "./rasterPixels";
import JSZip from "jszip";
// Public streaming API is documented by JSZip but absent from its bundled
// JSZipObject declaration: https://stuk.github.io/jszip/documentation/api_zipobject/internal_stream.html
// Keep its actual browser API typed; never substitute eager async inflation.
declare module "jszip" {
  interface JSZipObject { internalStream(type: "uint8array"): JSZip.JSZipStreamHelper<Uint8Array>; }
}
import { MAX_PUBLICATION_BYTES, MAX_PUBLICATION_ENTRY, MAX_PUBLICATION_EXPANDED, MAX_PUBLICATION_XML } from "./publicationData";
import { stripPublicationDoctype } from "./publicationXml";

// Validate physical records before JSZip normalizes paths or allocates entries.
// Return CRCs so streaming validation can check content without checkCRC32's
// unbounded eager inflation. Neither declared sizes nor compression ratios are
// trusted as a bound on actual decompression.
function directory(bytes: ArrayBuffer): Map<string, number> {
  if (bytes.byteLength > MAX_PUBLICATION_BYTES) throw new RangeError("bytes");
  const v = new DataView(bytes), length = bytes.byteLength;
  if (length < 22 || v.getUint32(0, true) !== 0x04034b50) throw Error("invalid");
  let end = -1;
  for (let i = length - 22; i >= Math.max(0, length - 65557); i--) {
    if (v.getUint32(i, true) === 0x06054b50 && i + 22 + v.getUint16(i + 20, true) === length) { if (end >= 0) throw Error("invalid"); end = i; }
  }
  if (end < 0 || v.getUint16(end + 4, true) || v.getUint16(end + 6, true)) throw Error("invalid");
  const count = v.getUint16(end + 10, true), start = v.getUint32(end + 16, true), size = v.getUint32(end + 12, true);
  if (count > 4000 || count === 65535) throw new RangeError("entries");
  if (v.getUint16(end + 8, true) !== count || start + size !== end) throw Error("invalid");
  const names = new Map<string, number>(); let offset = start, expanded = 0;
  const decoder = new TextDecoder("utf-8", { fatal: true });
  for (let i = 0; i < count; i++) {
    if (offset + 46 > end || v.getUint32(offset, true) !== 0x02014b50) throw Error("invalid");
    const flags = v.getUint16(offset + 8, true), method = v.getUint16(offset + 10, true), uncompressed = v.getUint32(offset + 24, true);
    if (flags & 1) throw Error("encrypted");
    if (![0, 8].includes(method) || v.getUint16(offset + 34, true)) throw Error("invalid");
    expanded += uncompressed;
    if (uncompressed > MAX_PUBLICATION_ENTRY || expanded > MAX_PUBLICATION_EXPANDED) throw new RangeError("expanded");
    const nameLength = v.getUint16(offset + 28, true), next = offset + 46 + nameLength + v.getUint16(offset + 30, true) + v.getUint16(offset + 32, true);
    if (next > end) throw Error("invalid");
    const name = decoder.decode(new Uint8Array(bytes, offset + 46, nameLength));
    if (!name || name.startsWith("/") || (/[\\:]/.test(name) || Array.from(name).some(char => char.charCodeAt(0) < 32 || char.charCodeAt(0) === 127)) || name.split("/").some((part, n, all) => part === "." || part === ".." || (!part && n !== all.length - 1)) || name.length > 4096 || names.has(name)) throw Error("invalid");
    names.set(name, v.getUint32(offset + 16, true)); offset = next;
  }
  if (offset !== end) throw Error("invalid");
  return names;
}
const crcTable = Uint32Array.from({ length: 256 }, (_, value) => {
  let crc = value; for (let bit = 0; bit < 8; bit++) crc = crc & 1 ? 0xedb88320 ^ (crc >>> 1) : crc >>> 1; return crc >>> 0;
});

function checkXml(value: string) {
  value = stripPublicationDoctype(value);
  let depth = 0, count = 0;
  for (const match of value.matchAll(/<!--[\s\S]*?-->|<!\[CDATA\[[\s\S]*?\]\]>|<\?[\s\S]*?\?>|<\/?[A-Za-z_](?:"[^"]*"|'[^']*'|[^'">])*?>/g)) {
    const token = match[0];
    if (token.startsWith("<!") || token.startsWith("<?")) continue;
    if (++count > 50000) throw new RangeError("elements");
    if (token.startsWith("</")) depth--; else if (!token.endsWith("/>")) depth++;
    if (depth > 64) throw new RangeError("depth");
    if (depth < 0) throw Error("invalid");
  }
  if (depth) throw Error("invalid");
  return count;
}


export async function readPublicationPackage(bytes: ArrayBuffer): Promise<Record<string, Uint8Array>> {
  const expected = directory(bytes), zip = await JSZip.loadAsync(bytes), output: Record<string, Uint8Array> = Object.create(null) as Record<string, Uint8Array>;
  let total = 0, elements = 0, pixels = 0;
  for (const [name, file] of Object.entries(zip.files)) {
    if (!expected.has(name) || file.unsafeOriginalName && file.unsafeOriginalName !== name) throw Error("invalid");
    if (file.dir) continue;
    let size = 0, crc = 0xffffffff;
    const chunks: Uint8Array[] = [];
    await new Promise<void>((resolve, reject) => {
      const stream = file.internalStream("uint8array"); let stopped = false;
      stream.on("data", chunk => {
        if (stopped) return;
        size += chunk.length; total += chunk.length;
        if (size > MAX_PUBLICATION_ENTRY || total > MAX_PUBLICATION_EXPANDED) { stopped = true; stream.pause(); reject(new RangeError("expanded")); return; }
        for (const byte of chunk) crc = (crcTable[(crc ^ byte) & 255] ?? 0) ^ (crc >>> 8);
        chunks.push(chunk);
      }).on("error", reject).on("end", () => { if (!stopped) resolve(); }).resume();
    });
    if (((crc ^ 0xffffffff) >>> 0) !== expected.get(name)) throw Error("invalid");
    const value = new Uint8Array(size); let offset = 0; for (const chunk of chunks) { value.set(chunk, offset); offset += chunk.length; }
    if (/\.(xml|rels|xhtml|html|opf|ncx|svg)$/i.test(name)) {
      if (size > MAX_PUBLICATION_XML) throw new RangeError("xml");
      elements += checkXml(new TextDecoder("utf-8", { fatal: true }).decode(value));
      if (elements > 200000) throw new RangeError("elements");
    }
    pixels += rasterPixels(value, name);
    if (pixels > 64000000) throw new RangeError("pixels");
    output[name] = value;
  }
  return output;
}
