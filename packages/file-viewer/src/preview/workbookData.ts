import { checkArchiveContainer } from "./archiveContainer.js";
// Serializable reader model. No parser code is imported by the workspace bundle.
export const MAX_WORKBOOK_BYTES = 15 * 1024 * 1024;
export const MAX_WORKBOOK_ROWS = 2000;
export const MAX_WORKBOOK_COLUMNS = 100;
export const MAX_WORKBOOK_CELLS = 50000;
export const MAX_WORKBOOK_SHEETS = 30;
export const WORKBOOK_PAGE_SIZE = 40;
export interface WorkbookCell { address: string; row: number; column: number; display: string; raw: string; type: string; formula?: string; merge?: string; missingValue: boolean; omitted?: boolean }
export interface WorkbookSheet { name: string; visibility: number; range: string; rows: number; columns: number; startRow: number; startColumn: number; cells: WorkbookCell[]; limited: boolean }
export interface WorkbookData { sheets: WorkbookSheet[]; totalSheets: number; limited: boolean }
export type WorkbookResult = { data: WorkbookData; error?: never } | { data?: never; error: "invalid" | "encrypted" | "limit" | "unavailable" | "fetch" | "unsupported" };
export function columnName(column: number): string {
  let name = "";
  for (let n = column + 1; n > 0; n = Math.floor((n - 1) / 26)) name = String.fromCharCode(65 + (n - 1) % 26) + name;
  return name;
}
export function cellCoordinates(address: string): { row: number; column: number } | null {
  const match = /^\$?([a-z]{1,3})\$?([1-9]\d{0,6})$/i.exec(address.trim());
  if (!match) return null;
  let column = 0; for (const letter of (match[1] ?? "").toUpperCase()) column = column * 26 + letter.charCodeAt(0) - 64;
  const row = Number(match[2]) - 1;
  return column <= 16384 && row < 1048576 ? { row, column: column - 1 } : null;
}
/** Rows of a sheet that match `query` (all when empty), optionally leaving one row out (the header). */
export function workbookRows(sheet: WorkbookSheet, query: string, skipRow?: number): number[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return Array.from({ length: sheet.rows }, (_, i) => sheet.startRow + i).filter((row) => row !== skipRow);
  return [...new Set(sheet.cells.filter((cell) => cell.row !== skipRow && `${cell.address}\n${cell.display}\n${cell.formula ?? ""}`.toLowerCase().includes(needle)).map((cell) => cell.row))].sort((a, b) => a - b);
}

/** How the first row is read: guessed from the sheet, or the reader's own choice. */
export type HeaderMode = "auto" | "header" | "data";
const HEADER_MAX_CHARS = 80;
const HEADER_BODY_ROWS = 20;
const TYPED_CELLS = new Set(["n", "b", "d"]);

/**
 * Column labels from the sheet's first row, indexed from `startColumn`, or null
 * when the first row is data. In "auto" the first row must be a header by shape:
 * every column holds a distinct, short, plain text value (no formulas), and the
 * rows below hold numbers, booleans or dates somewhere, so text above typed
 * values reads as labels. A table of text under text is left as data; the reader
 * can still choose "header". Blank labels in a chosen header fall back to the
 * column letter.
 */
export function workbookHeader(sheet: WorkbookSheet, mode: HeaderMode): string[] | null {
  if (mode === "data" || sheet.rows < 2 || sheet.columns < 1) return null;
  const first = new Map(sheet.cells.filter((cell) => cell.row === sheet.startRow).map((cell) => [cell.column, cell]));
  const labels = Array.from({ length: sheet.columns }, (_, index) => {
    const cell = first.get(sheet.startColumn + index);
    return cell && cell.type === "s" && !cell.formula && !cell.omitted && !cell.missingValue ? cell.display.trim() : "";
  });
  if (mode === "header") return labels.map((label, index) => label || columnName(sheet.startColumn + index));
  const distinct = new Set(labels.map((label) => label.toLowerCase()));
  if (labels.some((label) => !label || label.length > HEADER_MAX_CHARS) || distinct.size !== labels.length) return null;
  const lastBodyRow = sheet.startRow + Math.min(sheet.rows - 1, HEADER_BODY_ROWS);
  const typed = sheet.cells.some((cell) => cell.row > sheet.startRow && cell.row <= lastBodyRow && !cell.omitted && !cell.missingValue && TYPED_CELLS.has(cell.type));
  return typed ? labels : null;
}
// Shared ZIP checks also protect Word conversion before inflation.
export function checkWorkbookContainer(bytes: ArrayBuffer): void {
  checkArchiveContainer(bytes, MAX_WORKBOOK_BYTES);
}
