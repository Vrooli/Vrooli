import { Logger } from './types.js';

/**
 * Creates a simple logger mimicking the Winston interface for basic logging needs.
 * @returns A logger instance that writes to console
 */
export function createLogger(): Logger {
    function log(level: string, message: string, ...meta: any[]) {
        const timestamp = new Date().toISOString();
        console.log(`[${timestamp}] [${level.toUpperCase()}] ${message}`, ...meta);
    };

    return {
        info: (message: string, ...meta: any[]) => log('info', message, ...meta),
        warn: (message: string, ...meta: any[]) => log('warn', message, ...meta),
        error: (message: string, ...meta: any[]) => log('error', message, ...meta),
        debug: (message: string, ...meta: any[]) => log('debug', message, ...meta),
    };
} 