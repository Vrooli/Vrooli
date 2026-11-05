import { MAX_PUBLICATION_BYTES, type PublicationError } from "./publicationData";
import { readPublicationPackage } from "./readPublicationPackage";
import { readPreviewBytes } from "./readPreviewBytes";

self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_PUBLICATION_BYTES);
    const magic = new Uint8Array(bytes, 0, Math.min(4, bytes.byteLength));
    if (magic[0] === 0xd0 && magic[1] === 0xcf) { self.postMessage({ error: "encrypted" }); return; }
    const entries = await readPublicationPackage(bytes), xml: Record<string, string> = {};
    const decoder = new TextDecoder();
    for (const [name, value] of Object.entries(entries)) {
      if (/^ppt\/(notesSlides\/notesSlide\d+\.xml|slides\/_rels\/slide\d+\.xml\.rels)$/.test(name)) xml[name] = decoder.decode(value);
    }
    if (!entries["ppt/presentation.xml"] || !entries["ppt/_rels/presentation.xml.rels"]) throw Error("invalid");
    self.postMessage({ bytes, xml }, { transfer: [bytes] });
  } catch (error) {
    const reason: PublicationError = error instanceof RangeError ? "limit" : error instanceof Error && error.message === "fetch" ? "fetch" : error instanceof Error && error.message === "encrypted" ? "encrypted" : "invalid";
    self.postMessage({ error: reason });
  }
};
