// Pure formatting + parsing helpers for the file-preview renderers. Kept out of
// the component modules so React fast-refresh stays happy and the logic is
// unit-testable in isolation.
// formatBytes renders a byte count as a compact human-readable size.
export function formatBytes(bytes) {
    if (!Number.isFinite(bytes) || bytes < 0)
        return "";
    if (bytes < 1024)
        return `${bytes} B`;
    const units = ["KB", "MB", "GB", "TB"];
    let value = bytes / 1024;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024;
        unit += 1;
    }
    return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`;
}
// parseDelimited parses CSV/TSV text into rows of cells. It handles quoted
// fields with embedded delimiters, escaped quotes (""), and newlines inside
// quotes. Pure and dependency-free.
export function parseDelimited(text, delimiter) {
    const rows = [];
    let row = [];
    let field = "";
    let inQuotes = false;
    for (let i = 0; i < text.length; i++) {
        const ch = text[i];
        if (inQuotes) {
            if (ch === '"') {
                if (text[i + 1] === '"') {
                    field += '"';
                    i++;
                }
                else {
                    inQuotes = false;
                }
            }
            else {
                field += ch;
            }
            continue;
        }
        if (ch === '"') {
            inQuotes = true;
        }
        else if (ch === delimiter) {
            row.push(field);
            field = "";
        }
        else if (ch === "\n") {
            row.push(field);
            rows.push(row);
            row = [];
            field = "";
        }
        else if (ch === "\r") {
            // swallow; \n handles the row break
        }
        else {
            field += ch;
        }
    }
    if (field !== "" || row.length > 0) {
        row.push(field);
        rows.push(row);
    }
    return rows;
}
