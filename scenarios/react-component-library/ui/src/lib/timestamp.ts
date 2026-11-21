/**
 * Normalise any timestamp shape an API can hand us into a Date.
 *
 * The Connect/proto JSON encoding, a plain ISO string, a REST integer and a
 * client `Timestamp` instance all appear in this app. Calling `.toDate()` on a
 * value that arrived as a string is a runtime crash, so every boundary uses
 * this one reader instead of assuming the generated type survived the wire.
 */
export function timestampToDate(value: unknown): Date | undefined {
  if (!value) return undefined;
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? undefined : value;
  if (typeof value === "string") {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date;
  }
  if (typeof value === "number") {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date;
  }
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    if (typeof record.toDate === "function") {
      const date = (record.toDate as () => Date).call(value);
      return Number.isNaN(date.getTime()) ? undefined : date;
    }
    if (typeof record.seconds === "number" || typeof record.seconds === "bigint") {
      const seconds = Number(record.seconds);
      if (!Number.isFinite(seconds)) return undefined;
      const date = new Date(seconds * 1000);
      return Number.isNaN(date.getTime()) ? undefined : date;
    }
  }
  return undefined;
}
