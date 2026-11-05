import { useEffect, useState } from "react";
import type { StructureResult } from "./structuredData";

export function useStructuredData(source: string, json: boolean, truncated: boolean) {
  const [result, setResult] = useState<StructureResult | null>(null);
  useEffect(() => {
    if (truncated) { setResult({ error: "limit" }); return; }
    setResult(null);
    let worker: Worker;
    try { worker = new Worker(new URL("./structuredData.worker.ts", import.meta.url), { type: "module" }); }
    catch { setResult({ error: "unsupported" }); return; }
    let active = true;
    const finish = (value: StructureResult) => { if (active) { setResult(value); active = false; } window.clearTimeout(timeout); worker.terminate(); };
    const timeout = window.setTimeout(() => { finish({ error: "limit" }); }, 4000);
    worker.onmessage = (event: MessageEvent<StructureResult>) => { finish(event.data); };
    worker.onerror = (event) => { event.preventDefault(); finish({ error: "invalid" }); };
    worker.postMessage({ source, json });
    return () => { active = false; window.clearTimeout(timeout); worker.terminate(); };
  }, [source, json, truncated]);
  return result;
}
