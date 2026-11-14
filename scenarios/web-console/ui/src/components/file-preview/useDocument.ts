import { useEffect, useState } from "react";
import { MAX_DOCUMENT_BYTES, type DocumentResult } from "./documentData";
export function useDocument(href: string, size: number, attempt: number) {
  const [result, setResult] = useState<DocumentResult | null>(null);
  useEffect(() => {
    setResult(null);
    if (size > MAX_DOCUMENT_BYTES) { setResult({ error: "limit" }); return; }
    let worker: Worker;
    try { worker = new Worker(new URL("./document.worker.ts", import.meta.url), { type: "module" }); }
    catch { setResult({ error: "unavailable" }); return; }
    let active = true;
    const finish = (value: DocumentResult) => { if (active) { setResult(value); active = false; } window.clearTimeout(timeout); worker.terminate(); };
    const timeout = window.setTimeout(() => { finish({ error: "limit" }); }, 10000);
    worker.onmessage = (event: MessageEvent<DocumentResult>) => { finish(event.data); };
    worker.onerror = (event) => { event.preventDefault(); finish({ error: "invalid" }); };
    try { worker.postMessage({ href }); } catch { finish({ error: "unavailable" }); }
    return () => { active = false; window.clearTimeout(timeout); worker.terminate(); };
  }, [href, size, attempt]);
  return result;
}
