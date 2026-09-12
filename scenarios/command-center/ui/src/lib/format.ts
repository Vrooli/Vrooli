export const formatClock = (date: Date): string =>
  date.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit" });

/** Keep large supporting figures readable without changing the registry's raw unit. */
export const displayFormat = (value: number | null, format?: string): string | undefined => {
  if (value === null || Math.abs(value) < 1_000) return format;
  if (!format || format === "integer") return "compact";
  if (format === "currency") return "currency.compact";
  return format;
};

export const formatCompactNumber = (value: number): string =>
  new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 }).format(value);
