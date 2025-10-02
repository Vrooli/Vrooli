/**
 * Simple logging utility for ecosystem-manager UI
 * Provides consistent logging with optional debug mode
 */

const isDevelopment = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';

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
