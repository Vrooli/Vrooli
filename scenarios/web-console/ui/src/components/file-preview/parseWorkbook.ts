import { read, utils, type CellObject } from "xlsx";
import { checkWorkbookContainer, MAX_WORKBOOK_CELLS, MAX_WORKBOOK_COLUMNS, MAX_WORKBOOK_ROWS, MAX_WORKBOOK_SHEETS, type WorkbookCell, type WorkbookResult } from "./workbookData";

export function parseWorkbook(bytes: ArrayBuffer): WorkbookResult {
  try {
    checkWorkbookContainer(bytes);
    // Reject text/HTML disguised as Excel. Both XLS/BIFF and OOXML remain supported.
    const magic = new Uint8Array(bytes, 0, Math.min(8, bytes.byteLength));
    if (!(magic[0] === 0x50 && magic[1] === 0x4b || magic[0] === 0xd0 && magic[1] === 0xcf || magic[0] === 9 && [0, 2, 4, 8].includes(magic[1] ?? -1))) return { error: "invalid" };
    const metadata = read(bytes, { type: "array", bookSheets: true, bookProps: true });
    const workbook = read(bytes, { type: "array", sheets: metadata.SheetNames.slice(0, MAX_WORKBOOK_SHEETS), sheetRows: MAX_WORKBOOK_ROWS, cellFormula: true, cellText: true, cellNF: true, cellHTML: false, cellStyles: false, bookVBA: false, bookDeps: false, bookFiles: false, cellDates: false, WTF: false });
    let remaining = MAX_WORKBOOK_CELLS, characters = 0;
    const sheets = metadata.SheetNames.slice(0, MAX_WORKBOOK_SHEETS).map((name, index) => {
      const source = workbook.Sheets[name] ?? {};
      const fullRef: unknown = source["!fullref"];
      const range = typeof fullRef === "string" ? fullRef : source["!ref"] ?? "";
      const full = range ? utils.decode_range(range) : null;
      const bounded = source["!ref"] ? utils.decode_range(source["!ref"]) : null;
      const startRow = bounded?.s.r ?? 0, startColumn = bounded?.s.c ?? 0;
      const columns = bounded ? Math.min(MAX_WORKBOOK_COLUMNS, bounded.e.c - startColumn + 1) : 0;
      const rows = bounded ? Math.max(0, Math.min(MAX_WORKBOOK_ROWS - startRow, bounded.e.r - startRow + 1, Math.floor(remaining / Math.max(1, columns)))) : 0;
      let limited = !!full && (!bounded || rows < full.e.r - full.s.r + 1 || columns < full.e.c - full.s.c + 1);
      const cells: WorkbookCell[] = [], merges = (source["!merges"] ?? []).slice(0, 5000);
      if ((source["!merges"]?.length ?? 0) > merges.length) limited = true;
      for (let row = startRow; row < startRow + rows; row++) {
        const rowMerges = merges.filter((merge) => merge.s.r <= row && merge.e.r >= row);
        for (let column = startColumn; column < startColumn + columns; column++) {
        const address = utils.encode_cell({ r: row, c: column });
        const original = source[address] as CellObject | undefined;
        const merge = rowMerges.find((entry) => entry.s.c <= column && entry.e.c >= column);
        if ((!original || original.t === "z" && !original.f) && !merge) continue;
        const cell: CellObject = original ?? { t: "z" };
        const display = cell.w ?? (cell.v === undefined ? "" : String(cell.v));
        const raw = cell.v === undefined ? "" : String(cell.v);
        const formula = cell.f ? `=${cell.f}` : undefined;
        characters += display.length + raw.length + (formula?.length ?? 0);
        if (characters > 4000000 || display.length > 100000 || raw.length > 100000 || (formula?.length ?? 0) > 100000) {
          limited = true; cells.push({ address, row, column, display: "", raw: "", type: "omitted", missingValue: true, omitted: true }); continue;
        }
        cells.push({ address, row, column, display, raw, type: cell.t, formula, merge: merge ? utils.encode_range(merge) : undefined, missingValue: cell.v === undefined });
      }
      }
      remaining -= rows * columns;
      const visibility: unknown = workbook.Workbook?.Sheets?.[index]?.Hidden;
      return { name, visibility: typeof visibility === "number" ? visibility : 0, range, rows, columns, startRow, startColumn, cells, limited };
    });
    if (!sheets.length) return { error: "invalid" };
    return { data: { sheets, totalSheets: metadata.SheetNames.length, limited: metadata.SheetNames.length > MAX_WORKBOOK_SHEETS || sheets.some((sheet) => sheet.limited) } };
  } catch (error) {
    return { error: error instanceof RangeError ? "limit" : /password|encrypt/i.test(String(error)) ? "encrypted" : "invalid" };
  }
}
