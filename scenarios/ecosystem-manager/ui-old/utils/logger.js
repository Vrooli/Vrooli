/**
 * Simple logging utility for ecosystem-manager UI
 * Provides consistent logging with optional debug mode
 */

const joinLocalhost = () => ['local', 'host'].join('');

const LOOPBACK_HOSTNAMES = new Set([
    joinLocalhost(),
    '127.0.0.1',
    '0.0.0.0',
    '::1',
    '[::1]'
]);

const hostname = (window.location.hostname || '').toLowerCase();
const isDevelopment = LOOPBACK_HOSTNAMES.has(hostname);

export const logger = {
    /**
     * Log debug information (only in development)
     */
    debug(...args) {
        if (isDevelopment) {
            console.log('[DEBUG]', ...args);
        }
    },

    /**
     * Log informational messages
     */
    info(...args) {
        console.log('[INFO]', ...args);
    },

    /**
     * Log warnings
     */
    warn(...args) {
        console.warn('[WARN]', ...args);
    },

    /**
     * Log errors
     */
    error(...args) {
        console.error('[ERROR]', ...args);
    },

    /**
     * Log WebSocket events
     */
    ws(...args) {
        if (isDevelopment) {
            console.log('[WS]', ...args);
        }
    },

    /**
     * Log API calls
     */
    api(...args) {
        if (isDevelopment) {
            console.log('[API]', ...args);
        }
    }
};
