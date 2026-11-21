import { parseDocument } from "./parseDocument.js";
import { MAX_DOCUMENT_BYTES, type DocumentResult } from "./documentData.js";
import { readPreviewBytes } from "./readPreviewBytes.js";
self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  let result: DocumentResult;
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_DOCUMENT_BYTES);
    result = await parseDocument(bytes);
  } catch (error) { result = { error: error instanceof RangeError ? "limit" : "fetch" }; }
  self.postMessage(result);
};
