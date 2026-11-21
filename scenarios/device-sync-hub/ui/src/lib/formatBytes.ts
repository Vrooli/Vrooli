import { formatNumber } from "../i18n/format";

const UNITS = ["B", "KB", "MB", "GB", "TB"] as const;

/**
 * Human-readable byte size. Accepts the proto `int64` `size_bytes` (a bigint)
 * and renders e.g. "1.4 MB". Uses the locale-aware number formatter so the
 * decimal separator follows the active locale.
 */
export function formatBytes(bytes: bigint | number): string {
  let value = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (!Number.isFinite(value) || value < 0) value = 0;
  let unit = 0;
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  const formatted = formatNumber(value, {
    maximumFractionDigits: unit === 0 ? 0 : 1,
  });
  return `${formatted} ${UNITS[unit]}`;
}
