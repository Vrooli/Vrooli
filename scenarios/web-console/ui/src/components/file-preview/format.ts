// Pure formatting + parsing helpers for the file-preview renderers. Kept out of
// the component modules so React fast-refresh stays happy and the logic is
// unit-testable in isolation.

// formatBytes renders a byte count as a compact human-readable size.
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "";
  if (bytes < 1024) return `${String(bytes)} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value < 10 ? value.toFixed(1) : String(Math.round(value))} ${units[unit] ?? ""}`;
}

// parseDelimited parses CSV/TSV text into rows of cells. It handles quoted
// fields with embedded delimiters, escaped quotes (""), and newlines inside
// quotes. Pure and dependency-free.
export function parseDelimited(text: string, delimiter: string): string[][] {
  return parseDelimitedDetailed(text, delimiter).rows;
}

export function parseDelimitedDetailed(text: string, delimiter: string): { rows: string[][]; malformed: boolean } {
  text = text.replace(/^\uFEFF/, "");
  const rows: string[][] = [];
  let row: string[] = [];
  let field = "";
  let inQuotes = false;
  let started = false;
  let closedQuote = false;
  let malformed = false;
  for (let i = 0; i < text.length; i++) {
    const ch = text.charAt(i);
    if (inQuotes) {
      if (ch === '"') {
        if (text[i + 1] === '"') {
          field += '"';
          i++;
        } else {
          inQuotes = false;
          closedQuote = true;
        }
      } else {
        field += ch;
      }
      continue;
    }
    if (ch === '"' && field === "" && !closedQuote) {
      inQuotes = true;
      started = true;
    } else if (ch === delimiter) {
      row.push(field);
      field = "";
      started = false;
      closedQuote = false;
    } else if (ch === "\n" || ch === "\r") {
      row.push(field);
      rows.push(row);
      row = [];
      field = "";
      started = false;
      closedQuote = false;
      if (ch === "\r" && text[i + 1] === "\n") i++;
    } else {
      if (closedQuote || ch === '"') malformed = true;
      field += ch;
      started = true;
    }
  }
  if (field !== "" || row.length > 0 || started) {
    row.push(field);
    rows.push(row);
  }
  return { rows, malformed: malformed || inQuotes };
}
