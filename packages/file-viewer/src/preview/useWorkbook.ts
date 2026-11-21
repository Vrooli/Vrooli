import { useEffect, useState } from "react";
import { useFileViewerHost } from "../i18n/context.js";
import { MAX_WORKBOOK_BYTES, type WorkbookResult } from "./workbookData.js";

/** Longest a workbook may take to parse before the viewer gives up. */
export const WORKBOOK_DEADLINE_MS = 10000;

/**
 * Parse the workbook at `href` with the host's spreadsheet engine. Reports
 * `unsupported` when the host supplied none, and never starts work for a file
 * already known to exceed the byte limit. Bump `attempt` to retry.
 */
export function useWorkbook(href: string, size: number, attempt: number) {
  const load = useFileViewerHost().loaders.spreadsheet;
  const [result, setResult] = useState<WorkbookResult | null>(null);
  useEffect(() => {
    setResult(null);
    if (!load) { setResult({ error: "unsupported" }); return; }
    if (size > MAX_WORKBOOK_BYTES) { setResult({ error: "limit" }); return; }
    const controller = new AbortController();
    let active = true;
    const finish = (value: WorkbookResult) => {
      if (!active) return;
      active = false;
      window.clearTimeout(timeout);
      controller.abort();
      setResult(value);
    };
    const timeout = window.setTimeout(() => { finish({ error: "limit" }); }, WORKBOOK_DEADLINE_MS);
    void load().then((engine) => engine.parse(href, controller.signal)).then(finish, () => { finish({ error: "unavailable" }); });
    return () => { active = false; window.clearTimeout(timeout); controller.abort(); };
  }, [href, size, attempt, load]);
  return result;
}
