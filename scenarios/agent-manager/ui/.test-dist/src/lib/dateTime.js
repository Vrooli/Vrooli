import { timestampDate } from "@bufbuild/protobuf/wkt";
const dateTimeFormatters = new Map();
function getDateTimeFormatter(options) {
    const key = JSON.stringify(options ?? {});
    const cached = dateTimeFormatters.get(key);
    if (cached)
        return cached;
    const formatter = new Intl.DateTimeFormat("en-US", options);
    dateTimeFormatters.set(key, formatter);
    return formatter;
}
export function toValidDate(value) {
    if (!value)
        return undefined;
    if (value instanceof Date) {
        return Number.isNaN(value.getTime()) ? undefined : value;
    }
    if (typeof value === "object" && "seconds" in value) {
        return timestampDate(value);
    }
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}
export function formatDateTime(value, options, fallback = "N/A") {
    const date = toValidDate(value);
    if (!date)
        return fallback;
    return getDateTimeFormatter(options).format(date);
}
export function formatStatsDateTime(value) {
    return formatDateTime(value, {
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
    }, "N/A");
}
export function formatStatsChartDate(value) {
    return formatDateTime(value, {
        month: "short",
        day: "numeric",
        hour: "numeric",
    }, "N/A");
}
export function formatStatsChartTime(value) {
    return formatDateTime(value, {
        hour: "numeric",
        minute: "2-digit",
    }, "N/A");
}
export function formatChartAxisByPreset(value, preset) {
    if (preset === "30d") {
        return formatDateTime(value, {
            month: "short",
            day: "numeric",
        }, "N/A");
    }
    if (preset === "7d") {
        return formatStatsChartDate(value);
    }
    return formatStatsChartTime(value);
}
export function formatRelativeTimeShort(value, options) {
    const date = toValidDate(value);
    if (!date)
        return options?.fallback ?? "N/A";
    const now = options?.now ?? new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSeconds = Math.floor(diffMs / 1000);
    const diffMinutes = Math.floor(diffSeconds / 60);
    const diffHours = Math.floor(diffMinutes / 60);
    const diffDays = Math.floor(diffHours / 24);
    if (diffSeconds < 60)
        return "just now";
    if (diffMinutes < 60)
        return `${diffMinutes}m ago`;
    if (diffHours < 24)
        return `${diffHours}h ago`;
    const fallbackAfterDays = options?.fallbackAfterDays;
    if (typeof fallbackAfterDays === "number" &&
        diffDays >= fallbackAfterDays) {
        const fallbackFormatter = options?.fallbackFormatter ?? ((d) => formatStatsDateTime(d));
        return fallbackFormatter(date);
    }
    return `${diffDays}d ago`;
}
export function formatStandardDateTime(value) {
    return formatDateTime(value, undefined, "N/A");
}
export function formatStandardRelativeTime(value) {
    return formatRelativeTimeShort(value, { fallback: "N/A" });
}
export function formatStatsRelativeTime(value) {
    return formatRelativeTimeShort(value, {
        fallbackAfterDays: 7,
        fallbackFormatter: (date) => formatStatsDateTime(date),
    });
}
