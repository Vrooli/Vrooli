import winston from 'winston';
import type { Config } from '../config';

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

export const logger = loggerInstance;

export function setLogger(newLogger: winston.Logger): void {
  loggerInstance = newLogger;
}
