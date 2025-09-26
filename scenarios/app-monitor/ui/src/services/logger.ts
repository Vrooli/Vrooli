/**
 * Logger Service for App Monitor
 * Provides environment-aware logging with different levels
 */

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  NONE = 4,
}

class Logger {
  private level: LogLevel;
  private isDevelopment: boolean;
  private logHistory: Array<{ timestamp: Date; level: LogLevel; message: string; data?: unknown }> = [];
  private maxHistorySize = 100;

  constructor() {
    this.isDevelopment = import.meta.env.DEV;
    this.level = this.isDevelopment ? LogLevel.DEBUG : LogLevel.INFO;
    
    // Allow runtime log level configuration
    const configuredLevel = import.meta.env.VITE_LOG_LEVEL;
    if (configuredLevel && LogLevel[configuredLevel as keyof typeof LogLevel] !== undefined) {
      this.level = LogLevel[configuredLevel as keyof typeof LogLevel];
    }
  }

  private shouldLog(level: LogLevel): boolean {
    return level >= this.level;
  }

  private addToHistory(level: LogLevel, message: string, data?: unknown) {
    this.logHistory.push({
      timestamp: new Date(),
      level,
      message,
      data,
    });

    // Keep history size limited
    if (this.logHistory.length > this.maxHistorySize) {
      this.logHistory.shift();
    }
  }

  private formatMessage(level: string, message: string): string {
    const timestamp = new Date().toISOString();
    return `[${timestamp}] [${level}] ${message}`;
  }

  debug(message: string, data?: unknown) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      const formatted = this.formatMessage('DEBUG', message);
      if (data) {
        console.debug(formatted, data);
      } else {
        console.debug(formatted);
      }
      this.addToHistory(LogLevel.DEBUG, message, data);
    }
  }

  info(message: string, data?: unknown) {
    if (this.shouldLog(LogLevel.INFO)) {
      const formatted = this.formatMessage('INFO', message);
      if (data) {
        console.info(formatted, data);
      } else {
        console.info(formatted);
      }
      this.addToHistory(LogLevel.INFO, message, data);
    }
  }

  warn(message: string, data?: unknown) {
    if (this.shouldLog(LogLevel.WARN)) {
      const formatted = this.formatMessage('WARN', message);
      if (data) {
        console.warn(formatted, data);
      } else {
        console.warn(formatted);
      }
      this.addToHistory(LogLevel.WARN, message, data);
    }
  }

  error(message: string, error?: unknown) {
    if (this.shouldLog(LogLevel.ERROR)) {
      const formatted = this.formatMessage('ERROR', message);
      if (error) {
        console.error(formatted, error);
      } else {
        console.error(formatted);
      }
      this.addToHistory(LogLevel.ERROR, message, error);
    }
  }

  /**
   * Performance logging helper
   */
  time(label: string) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.time(label);
    }
  }

  timeEnd(label: string) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.timeEnd(label);
    }
  }

  /**
   * Group related logs together
   */
  group(label: string) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.group(label);
    }
  }

  groupEnd() {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.groupEnd();
    }
  }

  /**
   * Get log history for debugging purposes
   */
  getHistory() {
    return [...this.logHistory];
  }

  /**
   * Clear log history
   */
  clearHistory() {
    this.logHistory = [];
  }

  /**
   * Set log level at runtime
   */
  setLevel(level: LogLevel) {
    this.level = level;
    this.info(`Log level changed to ${LogLevel[level]}`);
  }

  /**
   * API call logging helper
   */
  logAPICall(method: string, url: string, data?: unknown) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      this.debug(`API ${method.toUpperCase()} ${url}`, data);
    }
  }

  /**
   * API response logging helper
   */
  logAPIResponse(url: string, status: number, data?: unknown) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      const level = status >= 400 ? LogLevel.ERROR : LogLevel.DEBUG;
      const message = `API Response ${url} - Status: ${status}`;
      if (level === LogLevel.ERROR) {
        this.error(message, data);
      } else {
        this.debug(message, data);
      }
    }
  }

  /**
   * Performance metrics logging
   */
  logPerformance(operation: string, duration: number) {
    if (this.shouldLog(LogLevel.DEBUG)) {
      const message = `Performance: ${operation} completed in ${duration}ms`;
      if (duration > 1000) {
        this.warn(message);
      } else {
        this.debug(message);
      }
    }
  }
}

// Export singleton instance
export const logger = new Logger();

// Export for testing and advanced usage
export default Logger;
