export const SOURCE_PAGE_SIZE = 400;
export const MAX_SOURCE_MATCHES = 500;
export type LogLevel = "error" | "warn" | "info" | "debug" | "other";
export interface SourceLine { number: number; text: string; level: LogLevel }
export interface SourceMatch { line: number; start: number; end: number }
export function logLevel(text: string): LogLevel {
  // Recognize labels, not incidental words in a message or source identifier.
  const label = "(FATAL|PANIC|ERROR|WARN(?:ING)?|INFO|DEBUG|TRACE)";
  const explicit = new RegExp(`(?:["']?(?:level|severity)["']?\\s*[:=]\\s*["']?)${label}\\b`, "i").exec(text);
  const bracketed = new RegExp(`\\[${label}\\]`, "i").exec(text);
  const prefix = new RegExp(`^\\s*(?:\\d{4}-\\d{2}-\\d{2}[T ]\\d{2}:\\d{2}:\\d{2}(?:[.,]\\d+)?(?:Z|[+-]\\d{2}:?\\d{2})?\\s+)?${label}(?=$|[:\\s])`, "i").exec(text);
  const level = (explicit?.[1] ?? bracketed?.[1] ?? prefix?.[1])?.toLowerCase();
  return level === "fatal" || level === "panic" || level === "error" ? "error" : level?.startsWith("warn") ? "warn" : level === "info" ? "info" : level === "debug" || level === "trace" ? "debug" : "other";
}
export function sourceLines(content: string): SourceLine[] { return content.split("\n").map((text, index) => ({ number: index + 1, text, level: logLevel(text) })); }
export function isLogPath(path: string): boolean { return /(?:\.(?:log|out|err)|(?:stdout|stderr)(?:\.txt)?)$/i.test(path); }
export function searchSource(lines: SourceLine[], query: string): { matches: SourceMatch[]; limited: boolean } {
  if (!query) return { matches: [], limited: false };
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const pattern = new RegExp(escaped, "giu"), matches: SourceMatch[] = [];
  for (const line of lines) {
    pattern.lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(line.text))) {
      if (matches.length === MAX_SOURCE_MATCHES) return { matches, limited: true };
      matches.push({ line: line.number, start: match.index, end: match.index + match[0].length });
    }
  }
  return { matches, limited: false };
}
export function selectedSource(content: string, anchor: number, end: number): string {
  const lines = content.split("\n"), last = Math.max(anchor, end);
  const selected = lines.slice(Math.min(anchor, end) - 1, last).join("\n");
  // CR belongs to a CRLF separator, not to the final selected line. Retain
  // literal CR at EOF and every separator between selected lines.
  return last < lines.length && selected.endsWith("\r") ? selected.slice(0, -1) : selected;
}
