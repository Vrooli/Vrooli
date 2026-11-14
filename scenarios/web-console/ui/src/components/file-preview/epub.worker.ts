import { MAX_PUBLICATION_BYTES } from "./publicationData";
import { readPublicationPackage } from "./readPublicationPackage";
import { readPreviewBytes } from "./readPreviewBytes";
self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  try {
    const parts = await readPublicationPackage(await readPreviewBytes(event.data.href, MAX_PUBLICATION_BYTES));
    if (parts["META-INF/encryption.xml"]) { self.postMessage({ error: "encrypted" }); return; }
    self.postMessage({ parts }, { transfer: Object.values(parts).map(part => part.buffer) });
  } catch (error) { self.postMessage({ error: error instanceof RangeError ? "limit" : error instanceof Error && error.message === "fetch" ? "fetch" : error instanceof Error && error.message === "encrypted" ? "encrypted" : "invalid" }); }
};
