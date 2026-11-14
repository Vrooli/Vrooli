import { parseWorkbook } from "./parseWorkbook";
import { MAX_WORKBOOK_BYTES, type WorkbookResult } from "./workbookData";
import { readPreviewBytes } from "./readPreviewBytes";
self.onmessage = async (event: MessageEvent<{ href: string }>) => {
  let result: WorkbookResult;
  try {
    const bytes = await readPreviewBytes(event.data.href, MAX_WORKBOOK_BYTES);
    result = parseWorkbook(bytes);
  } catch (error) { result = { error: error instanceof RangeError ? "limit" : "fetch" }; }
  self.postMessage(result);
};
