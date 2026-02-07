import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";

export type DateInput = string | Date | Timestamp | undefined;

const dateTimeFormatters = new Map<string, Intl.DateTimeFormat>();

function getDateTimeFormatter(
  options?: Intl.DateTimeFormatOptions
): Intl.DateTimeFormat {
  const key = JSON.stringify(options ?? {});
  const cached = dateTimeFormatters.get(key);
  if (cached) return cached;
  const formatter = new Intl.DateTimeFormat("en-US", options);
  dateTimeFormatters.set(key, formatter);
  return formatter;
}

export function toValidDate(value: DateInput): Date | undefined {
  if (!value) return undefined;
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? undefined : value;
  }
  if (typeof value === "object" && "seconds" in value) {
    return timestampDate(value as Timestamp);
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}

export function formatDateTime(
  value: DateInput,
  options?: Intl.DateTimeFormatOptions,
  fallback = "N/A"
): string {
  const date = toValidDate(value);
  if (!date) return fallback;
  return getDateTimeFormatter(options).format(date);
}

export function formatStatsDateTime(value: DateInput): string {
  return formatDateTime(
    value,
    {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
    "N/A"
  );
}

export function formatStatsChartDate(value: DateInput): string {
  return formatDateTime(
    value,
    {
      month: "short",
      day: "numeric",
      hour: "numeric",
    },
    "N/A"
  );
}

export function formatStatsChartTime(value: DateInput): string {
  return formatDateTime(
    value,
    {
      hour: "numeric",
      minute: "2-digit",
    },
    "N/A"
  );
}

export function formatChartAxisByPreset(
  value: DateInput,
  preset: string
): string {
  if (preset === "30d") {
    return formatDateTime(
      value,
      {
        month: "short",
        day: "numeric",
      },
      "N/A"
    );
  }
  if (preset === "7d") {
    return formatStatsChartDate(value);
  }
  return formatStatsChartTime(value);
}

interface RelativeTimeOptions {
  now?: Date;
  fallback?: string;
  fallbackAfterDays?: number;
  fallbackFormatter?: (date: Date) => string;
}

export function formatRelativeTimeShort(
  value: DateInput,
  options?: RelativeTimeOptions
): string {
  const date = toValidDate(value);
  if (!date) return options?.fallback ?? "N/A";

  const now = options?.now ?? new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSeconds < 60) return "just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;

  const fallbackAfterDays = options?.fallbackAfterDays;
  if (
    typeof fallbackAfterDays === "number" &&
    diffDays >= fallbackAfterDays
  ) {
    const fallbackFormatter =
      options?.fallbackFormatter ?? ((d: Date) => formatStatsDateTime(d));
    return fallbackFormatter(date);
  }

  return `${diffDays}d ago`;
}

export function formatStandardDateTime(value: DateInput): string {
  return formatDateTime(value, undefined, "N/A");
}

export function formatStandardRelativeTime(value: DateInput): string {
  return formatRelativeTimeShort(value, { fallback: "N/A" });
}

export function formatStatsRelativeTime(value: DateInput): string {
  return formatRelativeTimeShort(value, {
    fallbackAfterDays: 7,
    fallbackFormatter: (date) => formatStatsDateTime(date),
  });
}
