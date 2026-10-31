import { checkArchiveContainer } from "./archiveContainer";
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
export type WorkbookResult = { data: WorkbookData; error?: never } | { data?: never; error: "invalid" | "encrypted" | "limit" | "unavailable" | "fetch" };
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
export function workbookRows(sheet: WorkbookSheet, query: string): number[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return Array.from({ length: sheet.rows }, (_, i) => sheet.startRow + i);
  return [...new Set(sheet.cells.filter((cell) => `${cell.address}\n${cell.display}\n${cell.formula ?? ""}`.toLowerCase().includes(needle)).map((cell) => cell.row))].sort((a, b) => a - b);
}
// Shared ZIP checks also protect Word conversion before inflation.
export function checkWorkbookContainer(bytes: ArrayBuffer): void {
  checkArchiveContainer(bytes, MAX_WORKBOOK_BYTES);
}
