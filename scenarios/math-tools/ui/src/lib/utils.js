import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
export function cn(...inputs) {
    return twMerge(clsx(inputs));
}
export function parseNumberList(value) {
    return value
        .split(/[\s,]+/u)
        .map(entry => entry.trim())
        .filter(Boolean)
        .map(entry => Number(entry))
        .filter(entry => Number.isFinite(entry));
}
export function safeJsonParse(value) {
    try {
        return JSON.parse(value);
    }
    catch (error) {
        console.warn('[math-tools-ui] Failed to parse JSON', error);
        return undefined;
    }
}
export function formatTimestamp(date) {
    if (!date) {
        return '—';
    }
    try {
        const instance = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date;
        if (Number.isNaN(instance.getTime())) {
            return '—';
        }
        return instance.toLocaleString();
    }
    catch (error) {
        console.warn('[math-tools-ui] Unable to format timestamp', error);
        return '—';
    }
}
export function stringify(value, fallback = '—') {
    try {
        if (value === undefined || value === null) {
            return fallback;
        }
        if (typeof value === 'string') {
            return value;
        }
        if (typeof value === 'number' || typeof value === 'boolean') {
            return String(value);
        }
        return JSON.stringify(value, null, 2);
    }
    catch (error) {
        console.warn('[math-tools-ui] Unable to stringify value', error);
        return fallback;
    }
}
