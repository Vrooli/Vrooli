import { useEffect, useState } from "react";
import { MAX_WORKBOOK_BYTES, type WorkbookResult } from "./workbookData";
export function useWorkbook(href: string, size: number, attempt: number) {
  const [result, setResult] = useState<WorkbookResult | null>(null);
  useEffect(() => {
    setResult(null);
    if (size > MAX_WORKBOOK_BYTES) { setResult({ error: "limit" }); return; }
    let worker: Worker;
    try { worker = new Worker(new URL("./workbook.worker.ts", import.meta.url), { type: "module" }); }
    catch { setResult({ error: "unavailable" }); return; }
    let active = true;
    const finish = (value: WorkbookResult) => { if (active) { setResult(value); active = false; } window.clearTimeout(timeout); worker.terminate(); };
    const timeout = window.setTimeout(() => { finish({ error: "limit" }); }, 10000);
    worker.onmessage = (event: MessageEvent<WorkbookResult>) => { finish(event.data); };
    worker.onerror = (event) => { event.preventDefault(); finish({ error: "invalid" }); };
    try { worker.postMessage({ href }); } catch { finish({ error: "unavailable" }); }
    return () => { active = false; window.clearTimeout(timeout); worker.terminate(); };
  }, [href, size, attempt]);
  return result;
}
