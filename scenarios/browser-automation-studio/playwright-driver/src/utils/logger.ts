import winston from 'winston';
import type { Config } from '../config';

/**
 * Log context categories for consistent signal prefixing.
 * Use these to ensure logs are easily searchable and categorized.
 */
export const LogContext = {
  SERVER: 'server',
  SESSION: 'session',
  BROWSER: 'browser',
  INSTRUCTION: 'instruction',
  RECORDING: 'recording',
  CLEANUP: 'cleanup',
  TELEMETRY: 'telemetry',
  HEALTH: 'health',
} as const;

export type LogContextType = (typeof LogContext)[keyof typeof LogContext];

export function createLogger(config: Config): winston.Logger {
  const format =
    config.logging.format === 'json'
      ? winston.format.combine(
          winston.format.timestamp(),
          winston.format.errors({ stack: true }),
          winston.format.json()
        )
      : winston.format.combine(
          winston.format.timestamp(),
          winston.format.colorize(),
          winston.format.errors({ stack: true }),
          winston.format.printf(
            ({ level, message, timestamp, ...meta }) =>
              `${timestamp as string} [${level}]: ${message as string} ${
                Object.keys(meta).length > 0 ? JSON.stringify(meta) : ''
              }`
          )
        );

  return winston.createLogger({
    level: config.logging.level,
    format,
    transports: [new winston.transports.Console()],
  });
}

// Default logger instance (can be replaced via setLogger)
let loggerInstance: winston.Logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.errors({ stack: true }),
    winston.format.json()
  ),
  transports: [new winston.transports.Console()],
});

/**
 * Logger proxy that always delegates to the current loggerInstance.
 * This ensures that setLogger updates are reflected in all module imports.
 */
export const logger: winston.Logger = new Proxy({} as winston.Logger, {
  get(_target, prop) {
    return (loggerInstance as unknown as Record<string | symbol, unknown>)[prop];
  },
});

export function setLogger(newLogger: winston.Logger): void {
  loggerInstance = newLogger;
}

/**
 * Create a scoped log message with consistent prefix.
 * Example: scopedLog('session', 'created') => 'session: created'
 */
export function scopedLog(context: LogContextType, event: string): string {
  return `${context}: ${event}`;
}

/**
 * Create a no-op logger for contexts that don't need logging (e.g., replay preview).
 * All methods are defined but do nothing.
 */
export function createNoOpLogger(): winston.Logger {
  return winston.createLogger({
    level: 'silent',
    silent: true,
    transports: [],
  });
}
