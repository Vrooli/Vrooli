/// <reference lib="webworker" />
import { MAX_FONT_BYTES, fontError } from "./fontData";
import { readPreviewBytes } from "./readPreviewBytes";
import { inspectFontEnvelope, prepareFont } from "./prepareFont";
self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_FONT_BYTES);
    self.postMessage({ type: "parsing" });
    inspectFontEnvelope(bytes);
    // Fontkit's Brotli decoder can grow beyond its expected output. Require
    // native WOFF2/OTF sanitization before metadata parsing. This face is never
    // registered in a FontFaceSet; terminating the worker releases it.
    if (typeof FontFace === "undefined") throw new Error("unavailable");
    await new FontFace("wc-font-validation", bytes).load();
    const info = prepareFont(bytes);
    self.postMessage({ type: "ready", bytes, info }, { transfer: [bytes] });
  } catch (cause) { self.postMessage({ type: "error", error: fontError(cause) }); }
};
