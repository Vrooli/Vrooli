import { useEffect, useState } from "react";
import { MAX_FONT_BYTES, type FontError, type FontInfo } from "./fontData";
let nextFamily = 0;
export function useFontPreview(href: string, size: number, attempt: number) {
  const [loaded, setLoaded] = useState<{ family: string; info: FontInfo } | null>(null), [error, setError] = useState<FontError | null>(null);
  useEffect(() => {
    setLoaded(null); setError(null);
    if (size > MAX_FONT_BYTES) { setError("limit"); return; }
    if (typeof FontFace === "undefined" || typeof Reflect.get(document, "fonts") === "undefined") { setError("unavailable"); return; }
    let active = true, worker: Worker | undefined, face: FontFace | undefined;
    const stop = () => { active = false; worker?.terminate(); if (face) document.fonts.delete(face); };
    const fail = (reason: FontError) => { if (!active) return; clearTimeout(deadline); stop(); setLoaded(null); setError(reason); };
    let deadline = setTimeout(() => { fail("fetch"); }, 45000);
    try {
      worker = new Worker(new URL("./font.worker.ts", import.meta.url), { type: "module" });
      worker.onerror = () => { fail("unavailable"); };
      worker.onmessage = (event: MessageEvent<{ type: "parsing" } | { type: "error"; error: FontError } | { type: "ready"; bytes: ArrayBuffer; info: FontInfo }>) => {
        if (!active) return;
        if (event.data.type === "parsing") { clearTimeout(deadline); deadline = setTimeout(() => { fail("limit"); }, 10000); return; }
        worker?.terminate();
        if (event.data.type === "error") { fail(event.data.error); return; }
        clearTimeout(deadline); deadline = setTimeout(() => { fail("unavailable"); }, 15000);
        const family = `wc-preview-font-${String(++nextFamily)}`, { bytes, info } = event.data;
        try {
          const pending = new FontFace(family, bytes); face = pending;
          void pending.load().then(() => {
            if (!active) return;
            document.fonts.add(pending); clearTimeout(deadline); setLoaded({ family, info });
          }).catch(() => { fail("invalid"); });
        } catch { fail("invalid"); }
      };
      worker.postMessage({ href });
    } catch { fail("unavailable"); }
    return () => { clearTimeout(deadline); stop(); };
  }, [href, size, attempt]);
  return { loaded, error };
}
