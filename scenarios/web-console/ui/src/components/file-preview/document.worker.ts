import { parseDocument } from "./parseDocument";
import { MAX_DOCUMENT_BYTES, type DocumentResult } from "./documentData";
import { readPreviewBytes } from "./readPreviewBytes";
self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  let result: DocumentResult;
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_DOCUMENT_BYTES);
    result = await parseDocument(bytes);
  } catch (error) { result = { error: error instanceof RangeError ? "limit" : "fetch" }; }
  self.postMessage(result);
};
