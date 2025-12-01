/**
 * Formatting Utilities for Test Genie
 * Pure functions for formatting dates, times, durations, data, and HTML
 */

import { STATUS_DESCRIPTORS } from './constants.js';

/**
 * Format a timestamp as relative time (e.g., "5m ago", "3h ago")
 * @param {Date|string|number} timestamp
 * @returns {string}
 */
export function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

    if (diffMins < 60) {
        return `${diffMins}m ago`;
    } else if (diffHours < 24) {
        return `${diffHours}h ago`;
    } else {
        return date.toLocaleDateString();
    }
}

/**
 * Format a value as a locale-specific date and time string
 * @param {Date|string|number} value
 * @returns {string}
 */
export function formatDateTime(value) {
    const date = parseDate(value);
    if (!date) {
        return '—';
    }
    return date.toLocaleString();
}

/**
 * Format a date range (e.g., "Jan 1 → Jan 31 2024")
 * @param {Date|string|number} startValue
 * @param {Date|string|number} endValue
 * @returns {string}
 */
export function formatDateRange(startValue, endValue) {
    const start = parseDate(startValue);
    const end = parseDate(endValue);

    if (!start || !end) {
        return '';
    }

    const sameDay = start.toDateString() === end.toDateString();
    const options = { month: 'short', day: 'numeric' };
    const startLabel = start.toLocaleDateString(undefined, options);
    const endLabel = end.toLocaleDateString(undefined, options);
    const startYear = start.getFullYear();
    const endYear = end.getFullYear();

    if (sameDay) {
        return `${startLabel} ${startYear}`;
    }

    if (startYear === endYear) {
        return `${startLabel} → ${endLabel} ${endYear}`;
    }

    return `${startLabel} ${startYear} → ${endLabel} ${endYear}`;
}

/**
 * Format duration in seconds to human-readable string (e.g., "2h 30m 15s")
 * @param {number} totalSeconds
 * @returns {string}
 */
export function formatDurationSeconds(totalSeconds) {
    if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) {
        return '—';
    }
    const seconds = Math.round(totalSeconds);
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = seconds % 60;
    const parts = [];
    if (hours) parts.push(`${hours}h`);
    if (minutes) parts.push(`${minutes}m`);
    if (!hours && (remainingSeconds || parts.length === 0)) {
        parts.push(`${remainingSeconds}s`);
    }
    return parts.join(' ');
}

/**
 * Format a number as a percentage with specified decimal digits
 * @param {number} value
 * @param {number} digits - Number of decimal places
 * @returns {string}
 */
export function formatPercent(value, digits = 1) {
    if (!Number.isFinite(value)) {
        return '0%';
    }
    return `${value.toFixed(digits)}%`;
}

/**
 * Format phase name with proper capitalization (e.g., "test_phase" → "Test Phase")
 * @param {string} phase
 * @returns {string}
 */
export function formatPhaseLabel(phase) {
    if (!phase) {
        return 'Phase';
    }
    return phase
        .toString()
        .replace(/[_-]+/g, ' ')
        .split(' ')
        .filter(Boolean)
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' ');
}

/**
 * Format relative time with past/future support (e.g., "5m ago", "in 2h")
 * @param {Date|string|number} timestamp
 * @returns {string}
 */
export function formatRelativeTime(timestamp) {
    if (!timestamp) {
        return '';
    }

    const reference = timestamp instanceof Date ? timestamp : new Date(timestamp);
    if (Number.isNaN(reference.getTime())) {
        return '';
    }

    const now = Date.now();
    const diffMs = reference.getTime() - now;
    const absDiff = Math.abs(diffMs);
    const minute = 60 * 1000;
    const hour = 60 * minute;
    const day = 24 * hour;
    const isFuture = diffMs > 0;

    if (absDiff < minute) {
        return isFuture ? 'in <1m' : 'just now';
    }
    if (absDiff < hour) {
        const mins = Math.round(absDiff / minute);
        return `${mins}m ${isFuture ? 'from now' : 'ago'}`;
    }
    if (absDiff < day) {
        const hours = Math.round(absDiff / hour);
        return `${hours}h ${isFuture ? 'from now' : 'ago'}`;
    }
    if (absDiff < day * 7) {
        const days = Math.round(absDiff / day);
        return `${days}d ${isFuture ? 'from now' : 'ago'}`;
    }

    return reference.toLocaleDateString();
}

/**
 * Format timestamp with both absolute and relative time (e.g., "Jan 1, 2024 at 10:30 AM (5m ago)")
 * @param {Date|string|number} timestamp
 * @returns {string}
 */
export function formatDetailedTimestamp(timestamp) {
    if (!timestamp) {
        return 'unknown time';
    }

    const reference = timestamp instanceof Date ? timestamp : new Date(timestamp);
    if (Number.isNaN(reference.getTime())) {
        return 'unknown time';
    }

    const absolute = reference.toLocaleString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });

    const relative = formatRelativeTime(reference);
    return relative ? `${absolute} (${relative})` : absolute;
}

/**
 * Generic label formatter - converts snake_case/kebab-case to Title Case
 * @param {string} text
 * @returns {string}
 */
export function formatLabel(text) {
    if (!text) {
        return 'Unknown';
    }

    return String(text)
        .replace(/[_-]+/g, ' ')
        .split(' ')
        .filter(Boolean)
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ');
}

/**
 * Parse a value into a Date object
 * @param {Date|string|number} value
 * @returns {Date|null}
 */
export function parseDate(value) {
    if (!value) {
        return null;
    }
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? null : date;
}

/**
 * Normalize API response data into an array
 * @param {*} data - API response data
 * @param {string} primaryKey - Key to extract array from
 * @returns {Array}
 */
export function normalizeCollection(data, primaryKey) {
    if (!data) return [];
    if (Array.isArray(data)) {
        return data;
    }
    if (primaryKey && Array.isArray(data[primaryKey])) {
        return data[primaryKey];
    }
    if (Array.isArray(data.items)) {
        return data.items;
    }
    if (Array.isArray(data.data)) {
        return data.data;
    }
    return [];
}

/**
 * Normalize an ID value to a string
 * @param {*} value
 * @returns {string}
 */
export function normalizeId(value) {
    if (value === null || value === undefined) {
        return '';
    }
    return String(value);
}

/**
 * Convert value to a string representation (handles objects, arrays, etc.)
 * @param {*} value
 * @returns {string}
 */
export function stringifyValue(value) {
    if (value === null || value === undefined) {
        return '';
    }

    if (typeof value === 'string') {
        return value;
    }

    if (typeof value === 'number' || typeof value === 'boolean') {
        return String(value);
    }

    try {
        return JSON.stringify(value);
    } catch (error) {
        return String(value);
    }
}

/**
 * Escape HTML special characters to prevent XSS
 * @param {*} value
 * @returns {string}
 */
export function escapeHtml(value) {
    if (value === null || value === undefined) {
        return '';
    }

    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

/**
 * Clamp percentage value between 0 and 100
 * @param {number} value
 * @returns {number}
 */
export function clampPercentage(value) {
    if (!Number.isFinite(value)) {
        return 0;
    }
    const bounded = Math.max(0, Math.min(100, value));
    return Math.round(bounded);
}

/**
 * Calculate duration in seconds between two timestamps
 * @param {Date|string|number} startTime
 * @param {Date|string|number} endTime
 * @returns {number} Duration in seconds
 */
export function calculateDuration(startTime, endTime) {
    if (!startTime) return 0;
    if (!endTime) {
        // For running executions, calculate current duration
        const start = new Date(startTime);
        const now = new Date();
        return Math.round((now - start) / 1000 * 10) / 10; // Round to 1 decimal
    }
    const start = new Date(startTime);
    const end = new Date(endTime);
    return Math.round((end - start) / 1000 * 10) / 10; // Round to 1 decimal
}

/**
 * Get status descriptor object with key, label, and icon
 * @param {string} status
 * @returns {{key: string, label: string, icon: string}}
 */
export function getStatusDescriptor(status) {
    const normalized = (status || '').toString().toLowerCase();

    if ([
        'completed',
        'active',
        'success',
        'ready',
        'passed',
        'done'
    ].includes(normalized)) {
        return STATUS_DESCRIPTORS.completed;
    }

    if ([
        'running',
        'in_progress',
        'active_run',
        'ongoing',
        'processing'
    ].includes(normalized)) {
        return STATUS_DESCRIPTORS.running;
    }

    if ([
        'failed',
        'error',
        'blocked',
        'cancelled',
        'aborted'
    ].includes(normalized)) {
        return STATUS_DESCRIPTORS.failed;
    }

    return STATUS_DESCRIPTORS.pending;
}

/**
 * Extract suite types from suite and test cases
 * @param {Object} suite
 * @param {Array} testCases
 * @returns {Array<string>}
 */
export function extractSuiteTypes(suite, testCases) {
    const types = new Set();

    if (Array.isArray(suite.test_types)) {
        suite.test_types.filter(Boolean).forEach(type => types.add(type));
    }

    const rawSuiteType = suite.suite_type || suite.suiteType;
    if (typeof rawSuiteType === 'string') {
        rawSuiteType.split(',').map(token => token.trim()).filter(Boolean).forEach(type => types.add(type));
    }

    testCases.forEach(testCase => {
        if (testCase && testCase.test_type) {
            types.add(testCase.test_type);
        }
    });

    return Array.from(types);
}

/**
 * Count items by key and return count object
 * @param {Array} items
 * @returns {Object} Key-count pairs
 */
export function countBy(items) {
    const result = {};
    items.forEach(item => {
        const key = (item || 'unspecified').toString().toLowerCase();
        if (!result[key]) {
            result[key] = 0;
        }
        result[key] += 1;
    });
    return result;
}

/**
 * Summarize counts into a human-readable string
 * @param {Object} counts - Key-count pairs
 * @param {string} context - Context label for no data
 * @returns {string}
 */
export function summarizeCounts(counts, context = 'items') {
    const entries = Object.entries(counts || {}).filter(([, value]) => value > 0);

    if (entries.length === 0) {
        return `No ${context} data`;
    }

    entries.sort((a, b) => b[1] - a[1]);
    return entries.map(([key, value]) => `${formatLabel(key)} (${value})`).join(' • ');
}

/**
 * Normalize coverage target to valid range (50-100)
 * @param {number} value
 * @param {number} fallback - Default value if invalid
 * @returns {number}
 */
export function normalizeCoverageTarget(value, fallback = 80) {
    const num = Number(value);
    if (!Number.isFinite(num)) {
        return fallback;
    }
    return Math.min(100, Math.max(50, Math.round(num)));
}

/**
 * Normalize phase list to an array
 * @param {*} phases
 * @returns {Array<string>}
 */
export function normalizePhaseList(phases) {
    if (Array.isArray(phases)) {
        return phases.filter(Boolean).map(String);
    }
    if (phases instanceof Set) {
        return Array.from(phases).filter(Boolean).map(String);
    }
    if (typeof phases === 'string') {
        return phases.split(',').map(s => s.trim()).filter(Boolean);
    }
    return [];
}


/**
 * Get status CSS class
 * @param {string} status - Status string
 * @returns {string} CSS class name ('success', 'info', 'error', 'warning')
 */
export function getStatusClass(status) {
    const normalized = (status || '').toString().toLowerCase();

    if (['completed', 'active', 'success', 'ready', 'passed'].includes(normalized)) {
        return 'success';
    }

    if (['running', 'in_progress', 'queued', 'pending', 'scheduled'].includes(normalized)) {
        return 'info';
    }

    if (['failed', 'error', 'blocked', 'cancelled', 'aborted'].includes(normalized)) {
        return 'error';
    }

    return 'warning';
}
