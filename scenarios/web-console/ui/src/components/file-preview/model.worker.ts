/// <reference lib="webworker" />
import { MAX_MODEL_BYTES, modelError, type ModelFormat } from "./modelData";
import { readPreviewBytes } from "./readPreviewBytes";
import { prepareModel } from "./prepareModel";
self.onmessage = async (event: MessageEvent<{ href: string; format: ModelFormat }>) => {
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_MODEL_BYTES);
    self.postMessage({ type: "parsing" });
    const data = prepareModel(bytes, event.data.format);
    const transfer = "bytes" in data ? [data.bytes] : [data.positions.buffer, ...(data.normals ? [data.normals.buffer] : [])];
    self.postMessage({ type: "ready", data }, { transfer });
  } catch (error) { self.postMessage({ type: "error", error: modelError(error) }); }
};
