import type { SpreadsheetEngine } from "../loaders.js";
import type { WorkbookResult } from "../preview/workbookData.js";

/**
 * Parses workbooks in a module worker with SheetJS (`xlsx`, an optional peer
 * dependency of this package). Importing this subpath is what pulls `xlsx` into
 * a host's bundle, so a host opts in with
 * `loaders={{ spreadsheet: () => import("@vrooli/file-viewer/spreadsheet").then((m) => m.spreadsheetEngine) }}`.
 */
export const spreadsheetEngine: SpreadsheetEngine = {
  parse(href, signal) {
    return new Promise((resolve, reject) => {
      if (signal.aborted) { reject(signal.reason); return; }
      let worker: Worker;
      try { worker = new Worker(new URL("./workbook.worker.js", import.meta.url), { type: "module" }); }
      catch { resolve({ error: "unavailable" }); return; }
      const settle = (finish: () => void) => { signal.removeEventListener("abort", onAbort); worker.terminate(); finish(); };
      const onAbort = () => { settle(() => { reject(signal.reason); }); };
      signal.addEventListener("abort", onAbort, { once: true });
      worker.onmessage = (event: MessageEvent<WorkbookResult>) => { settle(() => { resolve(event.data); }); };
      worker.onerror = (event) => { event.preventDefault(); settle(() => { resolve({ error: "invalid" }); }); };
      try { worker.postMessage({ href }); } catch { settle(() => { resolve({ error: "unavailable" }); }); }
    });
  },
};
