export const MAX_TABLE_ROWS = 1000;
export const MAX_TABLE_COLUMNS = 200;
export const TABLE_PAGE_SIZE = 50;
export interface TableRow { index: number; cells: string[] }
export interface TableSort { column: number; direction: "asc" | "desc" }

export function prepareTable(parsed: string[][], firstRowHeader: boolean) {
  const values = firstRowHeader ? parsed.slice(1) : parsed;
  const rows = values.slice(0, MAX_TABLE_ROWS).map((cells, index) => ({ index, cells }));
  const columns = Math.max(parsed[0]?.length ?? 0, ...rows.map((row) => row.cells.length));
  return { header: firstRowHeader ? parsed[0] ?? [] : [], rows, columns, totalRows: values.length, truncated: values.length > MAX_TABLE_ROWS };
}

const NUMBER = /^[+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?$/i;
export function filterAndSortRows(rows: TableRow[], query: string, sort: TableSort | null, locale: string): TableRow[] {
  const needle = query.trim().toLocaleLowerCase(locale);
  const filtered = needle ? rows.filter((row) => row.cells.some((cell) => cell.toLocaleLowerCase(locale).includes(needle))) : [...rows];
  if (!sort) return filtered;
  const collator = new Intl.Collator(locale, { numeric: true, sensitivity: "base" });
  return filtered.sort((left, right) => {
    const a = left.cells[sort.column] ?? "";
    const b = right.cells[sort.column] ?? "";
    if (!a && b) return 1;
    if (a && !b) return -1;
    const x = Number(a), y = Number(b);
    const numeric = NUMBER.test(a) && NUMBER.test(b) && Number.isFinite(x) && Number.isFinite(y) && Math.abs(x) <= Number.MAX_SAFE_INTEGER && Math.abs(y) <= Number.MAX_SAFE_INTEGER;
    const comparison = numeric ? x - y : collator.compare(a, b);
    return (sort.direction === "asc" ? comparison : -comparison) || left.index - right.index;
  });
}

export function nextTableSort(sort: TableSort | null, column: number): TableSort | null {
  if (sort?.column !== column) return { column, direction: "asc" };
  return sort.direction === "asc" ? { column, direction: "desc" } : null;
}

export function columnWidth(width: number): number {
  return Math.max(100, Math.min(640, width));
}
