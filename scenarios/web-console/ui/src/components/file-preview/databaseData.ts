export const MAX_DATABASE_BYTES = 32 * 1024 * 1024;
export const DATABASE_PAGE_SIZE = 50;
export const DATABASE_MAX_PAGE = 1999;
export type DatabaseError = "invalid" | "limit" | "fetch" | "stale" | "busy" | "timeout" | "unavailable";
export type DatabaseCell = {
  type: "null" | "integer" | "real" | "text" | "blob";
  value: string;
  bytes: number;
  truncated: boolean;
};
export type DatabaseColumn = {
  name: string;
  type: string;
  required: boolean;
  primaryKey: number;
  defaultValue: string | null;
  generated: boolean;
};
export type DatabaseIndex = {
  name: string;
  unique: boolean;
  columns: string[];
  partial: boolean;
};
export type DatabaseForeignKey = {
  from: string;
  table: string;
  to: string;
};
export type DatabaseSchema = {
  columns: DatabaseColumn[];
  indexes: DatabaseIndex[];
  foreignKeys: DatabaseForeignKey[];
};
export type DatabaseInfo = {
  tables: string[];
  snapshotTime: string;
  bytes: number;
};
export type DatabaseQuery = {
  table: string;
  page: number;
  filterColumn: string;
  filter: string;
  sortColumn: string;
  descending: boolean;
};
export type DatabasePage = {
  query: DatabaseQuery;
  schema: DatabaseSchema;
  rows: DatabaseCell[][];
  hasMore: boolean;
  page: number;
};
export type DatabaseRequest = {
  type: "open";
  href: string;
} | {
  type: "page";
  id: number;
  query: DatabaseQuery;
};
export type DatabaseReply = {
  type: "loading";
} | {
  type: "ready";
  info: DatabaseInfo;
} | {
  type: "page";
  id: number;
  page: DatabasePage;
} | {
  type: "error";
  error: DatabaseError;
  id?: number;
};
export function quoteIdentifier(value: string) {
  return `"${value.replace(/"/g, '""')}"`;
}
export function inspectDatabaseHeader(bytes: ArrayBuffer) {
  const view = new DataView(bytes);
  if (bytes.byteLength < 100 || new TextDecoder().decode(bytes.slice(0, 16)) !== "SQLite format 3\0")
    throw new Error("invalid");
  if (bytes.byteLength > MAX_DATABASE_BYTES)
    throw new Error("limit");
  const encoded = view.getUint16(16), pageSize = encoded === 1 ? 65536 : encoded, pages = view.getUint32(28);
  if (pageSize < 512 || pageSize > 65536 || (pageSize & (pageSize - 1)) !== 0 || bytes.byteLength % pageSize !== 0 || pages * pageSize > MAX_DATABASE_BYTES)
    throw new Error("invalid");
  // Native snapshots are rollback-mode standalone images. An archive member
  // retaining a WAL header cannot be reconstructed without its matching journal.
  if (view.getUint8(18) !== 1 || view.getUint8(19) !== 1)
    throw new Error("invalid");
}
export function databaseCellText(cell: DatabaseCell) {
  return cell.type === "null" ? "NULL" : cell.type === "blob" ? `${String(cell.bytes)} B` : cell.value === "" ? '""' : cell.value;
}
export function databaseDetail(cell: DatabaseCell) {
  if (cell.type !== "text" || cell.truncated || cell.value.length > 65536)
    return databaseCellText(cell);
  try {
    const json: unknown = JSON.parse(cell.value);
    if (json !== null && typeof json === "object")
      return JSON.stringify(json, null, 2);
  }
  catch {
    // Ordinary text remains readable when it is not JSON.
  }
  return databaseCellText(cell);
}
function csvValue(value: string, escapeFormula = true) {
  const safe = escapeFormula && (/^[\s\uFEFF]*[=+@-]/u.test(value) || /^[\t\r\n]/u.test(value)) ? `'${value}` : value;
  return `"${safe.replace(/"/g, '""')}"`;
}
// Only the current bounded page is exported; truncation is explicit in the file.
export function databaseCsv(page: DatabasePage) {
  const header = page.schema.columns.map(column => csvValue(column.name)).join(',');
  const rows = page.rows.map(row => row.map(cell => {
    const value = databaseCellText(cell) + (cell.truncated ? '… [truncated]' : '');
    return csvValue(value, cell.type === 'text');
  }).join(','));
  return '\uFEFF' + [header, ...rows].join('\r\n');
}
