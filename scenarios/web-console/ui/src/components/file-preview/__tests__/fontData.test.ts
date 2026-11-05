import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { filterFontCharacters, fontError, fontHero, fontSample, fontFormat, MAX_FONT_BYTES, MAX_FONT_EXPANSION, unicodeLabel, visibleFontCharacter } from "../fontData";
import { inspectFontEnvelope, prepareFont } from "../prepareFont";
const fixture = (ext = "ttf") => Uint8Array.from(readFileSync(`../fixtures/file-preview/console-stencil.${ext}`)).buffer;
function table(bytes: ArrayBuffer, name: string) { const d = new DataView(bytes); for (let i = 0; i < d.getUint16(4); i++) { const offset = 12 + i * 16; if (String.fromCharCode(...new Uint8Array(bytes, offset, 4)) === name) return { offset, start: d.getUint32(offset + 8), size: d.getUint32(offset + 12) }; } throw Error(`No ${name}`); }
describe("bounded font metadata and coverage", () => {
  it.each(["ttf", "otf", "woff"])("reads original %s fixture and actual Unicode mappings", ext => {
    const info = prepareFont(fixture(ext)); expect(info.format).toBe(ext); expect(info.family).toBe("Console Stencil"); expect(info.style).toBe("Regular"); expect(info.characters).toContain(0x3a9); expect(info.characters).toContain(65); expect(info.characters).not.toContain(0x4e00); expect(info.axes).toEqual([]); expect(info.glyphs).toBeGreaterThan(60);
  });
  it("reads real WOFF2 with bounded transformed storage", () => {
    const bytes = Uint8Array.from(readFileSync("../../landing-page-business-suite/ui/public/presentation/jetbrains-mono.woff2")).buffer;
    const info = prepareFont(bytes); expect(info.format).toBe("woff2"); expect(info.characters).toContain(65); expect(info.family).toMatch(/JetBrains/i);
  });
  it("allows ordinary font license records within the total decoded-name budget", () => {
    const info = prepareFont(Uint8Array.from(readFileSync("../../../assets/public/sakbunderan-logo-only-webfont.woff2")).buffer);
    expect(info.characters).toContain(86); expect(info.characters).toContain(9724); expect(info.family).not.toContain(String.fromCharCode(127));
  });
  it("reads authored variable axes without fabricating styles", () => {
    const info = prepareFont(Uint8Array.from(readFileSync("../fixtures/file-preview/console-stencil-variable.ttf")).buffer);
    expect(info.axes).toEqual([{ tag: "wght", name: "Weight", min: 400, default: 400, max: 900 }]);
  });
  it("rejects oversized input before parsing", () => { expect(() => inspectFontEnvelope(new ArrayBuffer(MAX_FONT_BYTES + 1))).toThrow(RangeError); });
  it("rejects malformed, empty and collection signatures", () => {
    expect(() => prepareFont(new ArrayBuffer(0))).toThrow(); const bytes = fixture(); new Uint8Array(bytes).set([116,116,99,102]); expect(() => prepareFont(bytes)).toThrow("unsupported");
  });
  it("checks table count, duplicates and out-of-file storage", () => {
    let bytes = fixture(); new DataView(bytes).setUint16(4, 129); expect(() => prepareFont(bytes)).toThrow(RangeError);
    bytes = fixture(); new Uint8Array(bytes, 28, 4).set(new Uint8Array(bytes, 12, 4)); expect(() => prepareFont(bytes)).toThrow("invalid");
    bytes = fixture(); new DataView(bytes).setUint32(20, bytes.byteLength + 1); expect(() => prepareFont(bytes)).toThrow("invalid");
  });
  it("bounds compressed envelopes", () => {
    const bytes = fixture("woff"), d = new DataView(bytes); d.setUint32(16, MAX_FONT_EXPANSION + 1); expect(() => inspectFontEnvelope(bytes)).toThrow(RangeError);
    d.setUint32(16, 12); expect(() => inspectFontEnvelope(bytes)).toThrow("invalid");
  });
  it("bounds WOFF2 transformed allocation independently of advertised sfnt size", () => {
    const bytes = new ArrayBuffer(80), d = new DataView(bytes); new Uint8Array(bytes).set([119,79,70,50]); d.setUint32(4,0x10000); d.setUint32(8,80); d.setUint16(12,1); d.setUint32(16,1000); d.setUint32(20,1);
    new Uint8Array(bytes,48,7).set([10,1,0xa0,0x80,0x80,0x00,0]); expect(() => inspectFontEnvelope(bytes)).toThrow(RangeError);
  });
  it("rejects name count and duplicate-string amplification before decoding", () => {
    let bytes = fixture(); let name = table(bytes,"name"); new DataView(bytes).setUint16(name.start+2,513); expect(() => prepareFont(bytes)).toThrow(RangeError);
    bytes = fixture(); name = table(bytes,"name"); new DataView(bytes).setUint16(name.start+6+8,33000); expect(() => prepareFont(bytes)).toThrow(RangeError);
  });
  it("rejects out-of-file cmap pointers", () => {
    const bytes = fixture(), cmap = table(bytes,"cmap"), d = new DataView(bytes); d.setUint32(cmap.start+8,bytes.byteLength+1); expect(() => prepareFont(bytes)).toThrow();
  });
  it("filters mapped characters using text and Unicode prefixes", () => {
    const chars=[65,66,0x3a9,0x1f600]; expect(filterFontCharacters(chars,"ΩA")).toEqual([65,0x3a9]); expect(filterFontCharacters(chars,"U+004")).toEqual([65,66]); expect(filterFontCharacters(chars,"U+41")).toEqual([65]); expect(filterFontCharacters(chars,"0x1F600")).toEqual([0x1f600]); expect(filterFontCharacters(chars,"")).toEqual(chars); expect(unicodeLabel(65)).toBe("U+0041"); expect(visibleFontCharacter(0x202e)).toBe(false); expect(visibleFontCharacter(0x3a9)).toBe(true); expect(visibleFontCharacter(0xa0)).toBe(false); expect(visibleFontCharacter(0x2003)).toBe(false); expect(visibleFontCharacter(0xfe0f)).toBe(false); expect(visibleFontCharacter(0xe000)).toBe(true);
  });
  it("uses the actual font when the preferred specimen would entirely fall back", () => {
    const latin = Array.from("The quick brown fox.", char => char.codePointAt(0) ?? 0);
    expect(fontSample(latin, ["日本語", "The quick brown fox."])).toBe("The quick brown fox.");
    expect(fontSample([0xe000, 0xe001], ["Aa", "日本語"])).toBe("\ue000 \ue001");
    expect(fontHero([0xe000, 0xe001], "\ue000 \ue001")).toBe("\ue000\ue001"); expect(fontHero([65,97], "Sample")).toBe("Aa");
  });
  it("classifies errors and extensions", () => { expect(fontFormat("FACE.WOFF2")).toBe("woff2"); expect(fontFormat("a.ttc")).toBe(null); expect(fontError(new RangeError())).toBe("limit"); expect(fontError(new Error("fetch"))).toBe("fetch"); expect(fontError(new Error("unavailable"))).toBe("unavailable"); expect(fontError(new Error("bad data"))).toBe("invalid"); });
});
