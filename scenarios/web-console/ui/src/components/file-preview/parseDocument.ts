import mammoth from "mammoth/mammoth.browser";
import { checkArchiveContainer } from "./archiveContainer";
import { MAX_DOCUMENT_BYTES, MAX_DOCUMENT_HTML, MAX_DOCUMENT_IMAGES, type DocumentResult } from "./documentData";

// Runs in a disposable worker. Never use embedded style maps or external files.
export async function parseDocument(bytes: ArrayBuffer): Promise<DocumentResult> {
  try {
    if (bytes.byteLength > MAX_DOCUMENT_BYTES) return { error: "limit" };
    const magic = new Uint8Array(bytes, 0, Math.min(8, bytes.byteLength));
    if (magic[0] === 0xd0 && magic[1] === 0xcf) return { error: "encrypted" };
    if (magic[0] !== 0x50 || magic[1] !== 0x4b) return { error: "invalid" };
    checkArchiveContainer(bytes, MAX_DOCUMENT_BYTES, 64 * 1024 * 1024);
    let imageBytes = 0, omittedImages = 0;
    const result = await mammoth.convertToHtml({ arrayBuffer: bytes }, {
      externalFileAccess: false,
      includeEmbeddedStyleMap: false,
      idPrefix: "docx-",
      styleMap: ["p[style-name='Title'] => h1:fresh", "p[style-name='Subtitle'] => p:fresh"],
      convertImage: mammoth.images.imgElement(async (image) => {
        if (!/^image\/(png|jpeg|gif|webp)$/.test(image.contentType)) { omittedImages++; return { src: "" }; }
        const value = await image.readAsBase64String();
        imageBytes += value.length;
        if (value.length > 3 * 1024 * 1024 || imageBytes > MAX_DOCUMENT_IMAGES) { omittedImages++; return { src: "" }; }
        return { src: `data:${image.contentType};base64,${value}` };
      }),
    });
    if ((result.value.match(/<[a-z]/gi)?.length ?? 0) > 20000) return { error: "limit" };
    if (result.value.length > MAX_DOCUMENT_HTML + MAX_DOCUMENT_IMAGES) return { error: "limit" };
    // Bound prose independently from embedded raster data.
    if (result.value.replace(/src="data:[^"]*"/g, "").length > MAX_DOCUMENT_HTML) return { error: "limit" };
    return { data: { html: result.value, warnings: result.messages.length > 0, omittedImages } };
  } catch (error) { return { error: error instanceof RangeError ? "limit" : "invalid" }; }
}
