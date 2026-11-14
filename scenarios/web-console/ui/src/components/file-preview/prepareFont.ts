import { create } from "fontkit";
import { MAX_FONT_BYTES, MAX_FONT_CHARACTERS, MAX_FONT_EXPANSION, type FontFormat, type FontInfo } from "./fontData";

// Validate the storage envelope before the library allocates/decompresses any
// font tables. WOFF2 transformed lengths must be charged independently of the
// declared sfnt size (fontkit allocates from those directory values).
export function inspectFontEnvelope(bytes: ArrayBuffer): FontFormat {
  if (bytes.byteLength > MAX_FONT_BYTES) throw new RangeError("bytes");
  if (bytes.byteLength < 12) throw new Error("invalid");
  const view = new DataView(bytes), tag = (offset: number) => String.fromCharCode(...new Uint8Array(bytes, offset, 4));
  const signature = tag(0), web = signature === "wOFF" || signature === "wOF2";
  if (signature === "ttcf") throw new Error("unsupported");
  if (!web && signature !== "OTTO" && signature !== "true" && view.getUint32(0) !== 0x10000) throw new Error("invalid");
  const count = view.getUint16(web ? 12 : 4);
  if (!count || count > 128) throw new RangeError("tables");
  const tags = new Set<string>(); let expanded = 12 + 16 * count, allocated = 0;
  const charge = (name: string, length: number, stored = length) => {
    if (tags.has(name)) throw new Error("invalid"); tags.add(name);
    expanded += Math.ceil(length / 4) * 4; allocated += stored;
    if (expanded > MAX_FONT_EXPANSION || allocated > MAX_FONT_EXPANSION || (name === "fvar" && length > 65536) || (name === "name" && length > 1024 * 1024) || (name === "cmap" && length > 4 * 1024 * 1024)) throw new RangeError("tables");
  };
  if (web) {
    const header = signature === "wOF2" ? 48 : 44;
    if (bytes.byteLength < header || view.getUint32(8) !== bytes.byteLength || view.getUint16(14) !== 0) throw new Error("invalid");
    if (view.getUint32(4) === 0x74746366) throw new Error("unsupported");
    if (view.getUint32(16) > MAX_FONT_EXPANSION) throw new RangeError("expansion");
    if (signature === "wOF2") {
      const known = "cmap,head,hhea,hmtx,maxp,name,OS/2,post,cvt ,fpgm,glyf,loca,prep,CFF ,VORG,EBDT,EBLC,gasp,hdmx,kern,LTSH,PCLT,VDMX,vhea,vmtx,BASE,GDEF,GPOS,GSUB,EBSC,JSTF,MATH,CBDT,CBLC,COLR,CPAL,SVG ,sbix,acnt,avar,bdat,bloc,bsln,cvar,fdsc,feat,fmtx,fvar,gvar,hsty,just,lcar,mort,morx,opbd,prop,trak,Zapf,Silf,Glat,Gloc,Feat,Sill".split(",");
      let offset = header;
      const base128 = () => { let value = 0; for (let i = 0; i < 5; i++) { const byte = view.getUint8(offset++); if ((!i && byte === 0x80) || value > 0x1ffffff) throw new Error("invalid"); value = value * 128 + (byte & 127); if (!(byte & 128)) return value; } throw new Error("invalid"); };
      for (let i = 0; i < count; i++) {
        const flags = view.getUint8(offset++), index = flags & 63;
        const name = index === 63 ? tag(offset) : known[index]; if (index === 63) offset += 4;
        if (!name) throw new Error("invalid");
        const length = base128(), version = flags >> 6, transformed = name === "glyf" || name === "loca" ? version === 0 : version !== 0;
        const storage = transformed ? base128() : length; charge(name, length, storage);
      }
      if (!view.getUint32(20) || offset + view.getUint32(20) > bytes.byteLength) throw new Error("invalid");
    } else {
      if (header + count * 20 > bytes.byteLength) throw new Error("invalid");
      for (let i = 0; i < count; i++) {
        const offset = header + i * 20, start = view.getUint32(offset + 4), compressed = view.getUint32(offset + 8), length = view.getUint32(offset + 12);
        if (start < header + count * 20 || start + compressed > bytes.byteLength || compressed > length) throw new Error("invalid");
        charge(tag(offset), length);
      }
    }
    if (expanded > view.getUint32(16)) throw new Error("invalid");
  } else {
    if (12 + count * 16 > bytes.byteLength) throw new Error("invalid");
    for (let i = 0; i < count; i++) {
      const offset = 12 + i * 16, start = view.getUint32(offset + 8), length = view.getUint32(offset + 12);
      if (start < 12 + count * 16 || start + length > bytes.byteLength) throw new Error("invalid");
      charge(tag(offset), length);
    }
  }
  if (!["head", "maxp", "cmap", "name"].every(name => tags.has(name))) throw new Error("invalid");
  return signature === "wOF2" ? "woff2" : signature === "wOFF" ? "woff" : signature === "OTTO" ? "otf" : "ttf";
}
// The pinned fontkit adapter reads raw table streams before metadata getters.
// This internal API is deliberately confined here: it lets us bound name-string
// amplification and axis/cmap record counts before fontkit materializes them.
function validateMetadataTables(font: ReturnType<typeof create>) {
  const table = (tag: string) => {
    const record = font.directory.tables[tag], stream = font._getTableStream(tag);
    if (!record || !stream || stream.pos + record.length > stream.buffer.byteLength) throw new Error("invalid");
    return new DataView(stream.buffer.buffer, stream.buffer.byteOffset + stream.pos, record.length);
  };
  const name = table("name"), count = name.getUint16(2), strings = name.getUint16(4);
  if (count > 512 || name.getUint16(0) > 1) throw new RangeError("names");
  if (6 + count * 12 > name.byteLength || strings < 6 + count * 12 || strings > name.byteLength) throw new Error("invalid");
  let total = 0;
  for (let i = 0; i < count; i++) {
    const length = name.getUint16(6 + i * 12 + 8), offset = name.getUint16(6 + i * 12 + 10);
    total += length; if (length > 32768 || total > 65536) throw new RangeError("names");
    if (strings + offset + length > name.byteLength) throw new Error("invalid");
  }
  if (name.getUint16(0) === 1) {
    const offset = 6 + count * 12, languages = name.getUint16(offset);
    if (languages > 64 || offset + 2 + languages * 4 > name.byteLength) throw new RangeError("names");
    for (let i = 0; i < languages; i++) {
      const length = name.getUint16(offset + 2 + i * 4), start = name.getUint16(offset + 4 + i * 4);
      if (length > 256 || strings + start + length > name.byteLength) throw new Error("invalid");
    }
  }
  if (font.directory.tables.fvar) {
    const axes = table("fvar"), axisCount = axes.getUint16(8), axisSize = axes.getUint16(10), instances = axes.getUint16(12), instanceSize = axes.getUint16(14);
    if (axisCount > 16 || instances > 256) throw new RangeError("axes");
    if (axes.getUint16(4) !== 16 || axisSize !== 20 || instanceSize < 4 + axisCount * 4 || 16 + axisCount * axisSize + instances * instanceSize > axes.byteLength) throw new Error("invalid");
  }
  const cmap = table("cmap"), maps = cmap.getUint16(2);
  if (!maps || maps > 32) throw new RangeError("cmap");
  if (4 + maps * 8 > cmap.byteLength) throw new Error("invalid");
  for (let i = 0; i < maps; i++) {
    const offset = cmap.getUint32(4 + i * 8 + 4), version = cmap.getUint16(offset);
    if (![0, 4, 6, 10, 12, 13, 14].includes(version)) throw new Error("unsupported");
    const length = version === 14 ? cmap.getUint32(offset + 2) : version >= 10 ? cmap.getUint32(offset + 4) : cmap.getUint16(offset + 2);
    if (length < 6 || offset + length > cmap.byteLength) throw new Error("invalid");
    if (version === 12 || version === 13) {
      const groups = cmap.getUint32(offset + 12);
      if (groups > 65536 || 16 + groups * 12 > length) throw new RangeError("cmap");
      let last = -1;
      for (let j = 0; j < groups; j++) {
        const start = cmap.getUint32(offset + 16 + j * 12), end = cmap.getUint32(offset + 20 + j * 12);
        if (start <= last || start > end || end > 0x10ffff) throw new Error("invalid"); last = end;
      }
    }
    if (version === 4 && (cmap.getUint16(offset + 6) % 2 || 16 + cmap.getUint16(offset + 6) * 4 > length)) throw new Error("invalid");
  }
}
const label = (value: unknown) => typeof value === "string" ? Array.from(value).filter(char => { const code = char.charCodeAt(0); return code >= 32 && code !== 127 && !(code >= 0x202a && code <= 0x202e) && !(code >= 0x2066 && code <= 0x2069); }).join("").slice(0, 160) : "";
export function prepareFont(bytes: ArrayBuffer): FontInfo {
  const format = inspectFontEnvelope(bytes), font = create(new Uint8Array(bytes));
  if (!Number.isInteger(font.numGlyphs) || font.numGlyphs < 1 || font.numGlyphs > 65535 || !Number.isFinite(font.unitsPerEm) || font.unitsPerEm < 16 || font.unitsPerEm > 16384) throw new Error("invalid");
  validateMetadataTables(font);
  const characters: number[] = [];
  // A bounded scan avoids expanding a malicious cmap's declared ranges into an
  // unbounded array. hasGlyphForCodePoint reads mappings, never glyph outlines.
  for (let code = 0; code <= 0x10ffff; code++) {
    if (code >= 0xd800 && code <= 0xdfff) continue;
    if (font.hasGlyphForCodePoint(code)) { if (characters.length >= MAX_FONT_CHARACTERS) throw new RangeError("characters"); characters.push(code); }
  }
  const axes = Object.entries(font.variationAxes);
  if (axes.length > 16) throw new RangeError("axes");
  return { format, family: label(font.familyName), style: label(font.subfamilyName), glyphs: font.numGlyphs, unitsPerEm: font.unitsPerEm, characters, axes: axes.map(([tag, axis]) => {
    if (!/^[a-zA-Z0-9 ]{4}$/.test(tag) || ![axis.min, axis.default, axis.max].every(Number.isFinite) || axis.min > axis.default || axis.default > axis.max || Math.max(Math.abs(axis.min), Math.abs(axis.max)) > 1000000) throw new Error("invalid");
    return { tag, name: label(axis.name) || tag, min: axis.min, default: axis.default, max: axis.max };
  }) };
}
