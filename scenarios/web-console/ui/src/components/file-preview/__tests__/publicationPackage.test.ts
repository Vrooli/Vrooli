import JSZip from "jszip";
import { describe, expect, it } from "vitest";
import { readPublicationPackage } from "../readPublicationPackage";
import { MAX_PUBLICATION_ENTRY } from "../publicationData";
async function packageBytes(parts: Record<string, string | Uint8Array>, compressed = false) {
  const zip = new JSZip(); for (const [path, value] of Object.entries(parts)) zip.file(path, value);
  return zip.generateAsync({ type: "arraybuffer", compression: compressed ? "DEFLATE" : "STORE" });
}
function central(bytes: ArrayBuffer): number {
  const v = new DataView(bytes); for (let i = 0; i < bytes.byteLength - 4; i++) if (v.getUint32(i, true) === 0x02014b50) return i;
  throw Error("Missing directory");
}
describe("bounded publication packages", () => {
  it("retains exact UTF-8 XML and binary content from compressed packages", async () => {
    const bytes = await packageBytes({ "ppt/presentation.xml": '<presentation><title>مرحبا 日本語</title></presentation>', "ppt/media/content.bin": new Uint8Array([1, 2, 3]) }, true);
    const output = await readPublicationPackage(bytes);
    expect(new TextDecoder().decode(output["ppt/presentation.xml"])).toContain("مرحبا 日本語"); expect(output["ppt/media/content.bin"]).toEqual(new Uint8Array([1, 2, 3]));
  });
  it("rejects traversal, duplicate physical entries and encrypted flags before parsing", async () => {
    await expect(readPublicationPackage(await packageBytes({ "../escape.xml": "<x/>" }))).rejects.toThrow("invalid");
    const encrypted = await packageBytes({ "safe.xml": "<x/>" }); new DataView(encrypted).setUint16(central(encrypted) + 8, 1, true); await expect(readPublicationPackage(encrypted)).rejects.toThrow("encrypted");
    const duplicate = await packageBytes({ "one.xml": "<x/>", "two.xml": "<x/>" }); const v = new DataView(duplicate); let position = central(duplicate); position += 46 + v.getUint16(position + 28, true) + v.getUint16(position + 30, true) + v.getUint16(position + 32, true); new Uint8Array(duplicate, position + 46, 7).set(new TextEncoder().encode("one.xml")); await expect(readPublicationPackage(duplicate)).rejects.toThrow("invalid");
  });
  it("rejects checksum corruption and mismatched central record counts", async () => {
    const bytes = await packageBytes({ "safe.xml": "<x/>" }); const v = new DataView(bytes); v.setUint32(central(bytes) + 16, 123, true); await expect(readPublicationPackage(bytes)).rejects.toThrow("invalid");
    const count = await packageBytes({ "safe.xml": "<x/>" }); const cv = new DataView(count); cv.setUint16(count.byteLength - 22 + 10, 0, true); cv.setUint16(count.byteLength - 22 + 8, 0, true); await expect(readPublicationPackage(count)).rejects.toThrow("invalid");
  });
  it("stops actual decompression even when compressed headers lie about expansion", async () => {
    const bytes = await packageBytes({ "bomb.bin": new Uint8Array(MAX_PUBLICATION_ENTRY + 32768) }, true), v = new DataView(bytes);
    v.setUint32(central(bytes) + 24, 1, true); v.setUint32(22, 1, true);
    await expect(readPublicationPackage(bytes)).rejects.toBeInstanceOf(RangeError);
  }, 15000);
  it("rejects entity declarations and deeply nested XML", async () => {
    await expect(readPublicationPackage(await packageBytes({ "book.xml": '<!DOCTYPE x [<!ENTITY a "b">]><x>&a;</x>' }))).rejects.toThrow("invalid");
    await expect(readPublicationPackage(await packageBytes({ "book.xml": "<x>".repeat(65) + "</x>".repeat(65) }))).rejects.toBeInstanceOf(RangeError);
  });
  it("rejects raster decode bombs independently of compressed file size", async () => {
    const png = new Uint8Array(32), v = new DataView(png.buffer); v.setUint32(0, 0x89504e47); v.setUint32(4, 0x0d0a1a0a); v.setUint32(16, 100000); v.setUint32(20, 100000);
    await expect(readPublicationPackage(await packageBytes({ "OEBPS/large.png": png }, true))).rejects.toBeInstanceOf(RangeError);
  });
});
