import { parseWorkbook } from "./parseWorkbook.js";
import { MAX_WORKBOOK_BYTES, type WorkbookResult } from "../preview/workbookData.js";
import { readPreviewBytes } from "../preview/readPreviewBytes.js";
self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  let result: WorkbookResult;
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_WORKBOOK_BYTES);
    result = parseWorkbook(bytes);
  } catch (error) { result = { error: error instanceof RangeError ? "limit" : "fetch" }; }
  self.postMessage(result);
};
