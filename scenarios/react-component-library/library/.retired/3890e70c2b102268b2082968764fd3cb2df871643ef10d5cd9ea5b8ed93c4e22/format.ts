export interface FormattedFigure {
  /** Characters of the number itself, for digit-level animation. */
  text: string;
  prefix: string;
  suffix: string;
}

const compact = (value: number, digits = 1): string => {
  const abs = Math.abs(value);
  if (abs >= 1_000_000) return `${trim((value / 1_000_000).toFixed(digits))}M`;
  if (abs >= 10_000) return `${trim((value / 1_000).toFixed(digits))}k`;
  return group(value, 0);
};

const trim = (s: string): string => s.replace(/\.0+$/, "").replace(/(\.\d*?)0+$/, "$1");
const group = (value: number, maxFraction: number): string =>
  new Intl.NumberFormat("en-US", { maximumFractionDigits: maxFraction }).format(value);

/**
 * Formats a reading's number for display. The format vocabulary is the
 * registry's: integer, compact, currency, currency.compact, percent,
 * percent.signed, minutes, duration.days.
 */
export function formatFigure(value: number, format?: string, unit?: string): FormattedFigure {
  switch (format) {
    case "currency":
      return { prefix: "$", text: group(value, 0), suffix: "" };
    case "currency.compact":
      return { prefix: "$", text: compact(value), suffix: "" };
    case "percent":
      return { prefix: "", text: group(value * 100, value * 100 < 10 ? 1 : 0), suffix: "%" };
    case "percent.signed":
      return { prefix: value >= 0 ? "+" : "−", text: group(Math.abs(value) * 100, 1), suffix: "%" };
    case "minutes":
      return { prefix: "", text: group(value, value < 10 ? 1 : 0), suffix: " min" };
    case "duration.days":
      return { prefix: "", text: group(value, 1), suffix: " d" };
    case "compact":
      return { prefix: "", text: compact(value), suffix: "" };
    default:
      return { prefix: "", text: group(value, Number.isInteger(value) ? 0 : 1), suffix: unit && unit !== "count" ? ` ${unit}` : "" };
  }
}

export const formatClock = (date: Date): string =>
  date.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit" });