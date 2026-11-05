import { useEffect, useState } from "react";
import { MAX_PUBLICATION_BYTES, type PublicationError } from "./publicationData";
import { parseEbook, type EbookData } from "./epubData";
export function useEbook(href: string, size: number, attempt: number) {
  const [result, setResult] = useState<{ data: EbookData } | { error: PublicationError } | null>(null);
  useEffect(() => {
    setResult(null);
    if (size > MAX_PUBLICATION_BYTES) { setResult({ error: "limit" }); return; }
    let worker: Worker;
    try { worker = new Worker(new URL("./epub.worker.ts", import.meta.url), { type: "module" }); } catch { setResult({ error: "unavailable" }); return; }
    let active = true;
    const finish = (value: NonNullable<typeof result>) => { if (active) { active = false; setResult(value); } window.clearTimeout(timeout); worker.terminate(); };
    const timeout = window.setTimeout(() => { finish({ error: "limit" }); }, 10000);
    worker.onmessage = (event: MessageEvent<{ parts: EbookData["parts"] } | { error: PublicationError }>) => {
      if (!active) return;
      try { finish("error" in event.data ? event.data : { data: parseEbook(event.data.parts) }); }
      catch (error) { finish({ error: error instanceof RangeError ? "limit" : "invalid" }); }
    };
    worker.onerror = event => { event.preventDefault(); finish({ error: "invalid" }); };
    try { worker.postMessage({ href }); } catch { finish({ error: "unavailable" }); }
    return () => { active = false; window.clearTimeout(timeout); worker.terminate(); };
  }, [href, size, attempt]);
  return result;
}
