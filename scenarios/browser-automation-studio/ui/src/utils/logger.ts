/**
 * Centralized logging utility for Browser Automation Studio
 * Provides consistent log formatting and future extensibility for remote logging
 */

export enum LogLevel {
  DEBUG = 'DEBUG',
  INFO = 'INFO',
  WARN = 'WARN',
  ERROR = 'ERROR',
}

interface LogContext {
  component?: string;
  action?: string;
  [key: string]: unknown;
}

class Logger {
  private readonly prefix = '[BrowserAutomationStudio]';
  private readonly enabledLevels: Set<LogLevel>;

  constructor() {
    // In development, enable all levels; in production, only WARN and ERROR
    this.enabledLevels = new Set(
      import.meta.env.DEV
        ? [LogLevel.DEBUG, LogLevel.INFO, LogLevel.WARN, LogLevel.ERROR]
        : [LogLevel.WARN, LogLevel.ERROR]
    );
  }

  private formatMessage(level: LogLevel, message: string, context?: LogContext): string {
    const parts = [this.prefix, `[${level}]`];

    if (context?.component) {
      parts.push(`[${context.component}]`);
    }

    if (context?.action) {
      parts.push(`[${context.action}]`);
    }

    parts.push(message);

    return parts.join(' ');
  }

  private log(level: LogLevel, message: string, context?: LogContext, error?: Error | unknown): void {
    if (!this.enabledLevels.has(level)) {
      return;
    }

    const formattedMessage = this.formatMessage(level, message, context);

    switch (level) {
      case LogLevel.DEBUG:
      case LogLevel.INFO:
        // eslint-disable-next-line no-console
        console.log(formattedMessage, context && Object.keys(context).length > 0 ? context : '');
        break;
      case LogLevel.WARN:
        // eslint-disable-next-line no-console
        console.warn(formattedMessage, context && Object.keys(context).length > 0 ? context : '', error || '');
        break;
      case LogLevel.ERROR:
        // eslint-disable-next-line no-console
        console.error(formattedMessage, context && Object.keys(context).length > 0 ? context : '', error || '');
        break;
    }
  }

  debug(message: string, context?: LogContext): void {
    this.log(LogLevel.DEBUG, message, context);
  }

  info(message: string, context?: LogContext): void {
    this.log(LogLevel.INFO, message, context);
  }

  warn(message: string, context?: LogContext, error?: Error | unknown): void {
    this.log(LogLevel.WARN, message, context, error);
  }

  error(message: string, context?: LogContext, error?: Error | unknown): void {
    this.log(LogLevel.ERROR, message, context, error);
  }
}

// Export singleton instance
export const logger = new Logger();
