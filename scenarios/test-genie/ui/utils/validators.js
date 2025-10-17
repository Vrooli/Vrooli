/**
 * Validation Utilities for Test Genie
 * Pure functions for validating data and inputs
 */

import { DEFAULT_GENERATION_PHASES, COVERAGE } from './constants.js';

/**
 * Check if a health payload indicates the system is healthy
 * @param {*} payload - Health check payload
 * @returns {boolean}
 */
export function isHealthPayloadHealthy(payload) {
    if (!payload || typeof payload !== 'object') {
        return false;
    }
    if (typeof payload.healthy === 'boolean') {
        return payload.healthy;
    }
    const status = String(payload.status || '').toLowerCase();
    return status === 'healthy' || status === 'ok' || status === 'active';
}

/**
 * Normalize coverage target to valid range (50-100)
 * @param {*} value - Input value
 * @param {number} fallback - Default value if invalid
 * @returns {number}
 */
export function normalizeCoverageTarget(value, fallback = COVERAGE.DEFAULT_TARGET) {
    const numeric = Number.parseInt(value, 10);
    const base = Number.isFinite(numeric) ? numeric : Number.parseInt(fallback, 10);
    const valid = Number.isFinite(base) ? base : COVERAGE.DEFAULT_TARGET;
    return Math.min(COVERAGE.MAX_TARGET, Math.max(COVERAGE.MIN_TARGET, valid));
}

/**
 * Normalize phase list - validates against DEFAULT_GENERATION_PHASES
 * @param {Array|Set|string} phases - Input phases
 * @returns {Array<string>}
 */
export function normalizePhaseList(phases) {
    const allowList = new Set(DEFAULT_GENERATION_PHASES);
    if (!Array.isArray(phases)) {
        return DEFAULT_GENERATION_PHASES.slice();
    }
    const normalized = phases
        .filter(phase => phase && typeof phase === 'string' && allowList.has(phase))
        .map(String);
    if (!normalized.length) {
        return DEFAULT_GENERATION_PHASES.slice();
    }
    return Array.from(new Set(normalized));
}

/**
 * Validate suite ID format
 * @param {*} suiteId
 * @returns {boolean}
 */
export function isValidSuiteId(suiteId) {
    if (!suiteId) return false;
    const str = String(suiteId);
    return str.length > 0 && str.trim() !== '';
}

/**
 * Validate scenario name
 * @param {*} scenarioName
 * @returns {boolean}
 */
export function isValidScenarioName(scenarioName) {
    if (!scenarioName) return false;
    const str = String(scenarioName).trim();
    return str.length > 0 && str.length <= 255;
}

/**
 * Validate email address (basic validation)
 * @param {string} email
 * @returns {boolean}
 */
export function isValidEmail(email) {
    if (!email || typeof email !== 'string') return false;
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

/**
 * Validate URL format
 * @param {string} url
 * @returns {boolean}
 */
export function isValidUrl(url) {
    if (!url || typeof url !== 'string') return false;
    try {
        new URL(url);
        return true;
    } catch {
        return false;
    }
}

/**
 * Validate timeout value (must be positive number in seconds)
 * @param {*} timeout
 * @returns {boolean}
 */
export function isValidTimeout(timeout) {
    const num = Number(timeout);
    return Number.isFinite(num) && num > 0 && num <= 3600; // Max 1 hour
}

/**
 * Validate percentage value (0-100)
 * @param {*} percentage
 * @returns {boolean}
 */
export function isValidPercentage(percentage) {
    const num = Number(percentage);
    return Number.isFinite(num) && num >= 0 && num <= 100;
}

/**
 * Validate array is non-empty
 * @param {*} arr
 * @returns {boolean}
 */
export function isNonEmptyArray(arr) {
    return Array.isArray(arr) && arr.length > 0;
}

/**
 * Validate object is non-empty
 * @param {*} obj
 * @returns {boolean}
 */
export function isNonEmptyObject(obj) {
    return obj && typeof obj === 'object' && !Array.isArray(obj) && Object.keys(obj).length > 0;
}

/**
 * Sanitize user input for display (remove dangerous characters)
 * @param {string} input
 * @returns {string}
 */
export function sanitizeInput(input) {
    if (!input || typeof input !== 'string') return '';
    return input
        .replace(/[<>]/g, '') // Remove angle brackets
        .trim()
        .slice(0, 1000); // Limit length
}

/**
 * Validate JSON string
 * @param {string} jsonString
 * @returns {boolean}
 */
export function isValidJson(jsonString) {
    if (typeof jsonString !== 'string') return false;
    try {
        JSON.parse(jsonString);
        return true;
    } catch {
        return false;
    }
}

/**
 * Validate date is in the past
 * @param {Date|string|number} date
 * @returns {boolean}
 */
export function isPastDate(date) {
    const d = new Date(date);
    if (Number.isNaN(d.getTime())) return false;
    return d.getTime() < Date.now();
}

/**
 * Validate date is in the future
 * @param {Date|string|number} date
 * @returns {boolean}
 */
export function isFutureDate(date) {
    const d = new Date(date);
    if (Number.isNaN(d.getTime())) return false;
    return d.getTime() > Date.now();
}
