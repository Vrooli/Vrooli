import winston from "winston";

const LOG_DIR = `${process.env.PROJECT_DIR}/data/logs`;
const MAX_LOG_SIZE = 5_242_880; // 5MB

/**
 * @returns Array of transports to use for logging, depending on environment
 */
function getTransports() {
    const transports: winston.transport[] = [];
    const isTest = process.env.JEST_WORKER_ID !== undefined || process.env.NODE_ENV === "test";
    const canLogToFile = process.env.PROJECT_DIR !== undefined;

    // Add file transports when in development or production
    if (canLogToFile && !isTest) {
        transports.push(
            new winston.transports.File({
                filename: `${LOG_DIR}/error.log`,
                level: "error",
                maxsize: MAX_LOG_SIZE,
            }),
            new winston.transports.File({
                filename: `${LOG_DIR}/combined.log`,
                maxsize: MAX_LOG_SIZE,
            }),
        );
    }

    // Add console transports for development
    if (!(process.env.NODE_ENV || "").startsWith("prod")) {
        transports.push(
            new winston.transports.Console({
                format: winston.format.combine(
                    winston.format.errors({ stack: true }),
                    winston.format.simple(),
                ),
            }),
        );
    }

    return transports;
}

/**
 * Get the log level from environment variables
 * In tests, default to 'error' to reduce noise
 * Otherwise, default to 'info'
 */
function getLogLevel(): string {
    const envLevel = process.env.LOG_LEVEL?.toLowerCase();
    const isTest = process.env.JEST_WORKER_ID !== undefined || process.env.NODE_ENV === "test";
    
    // Valid winston syslog levels
    const validLevels = ["emerg", "alert", "crit", "error", "warning", "notice", "info", "debug"];
    
    if (envLevel && validLevels.includes(envLevel)) {
        return envLevel;
    }
    
    // Default to 'error' in tests, 'info' otherwise
    return isTest ? "error" : "info";
}

/**
 * Preferred logging method. Allows you to specify 
 * the level and output location(s). Includes timestamp 
 * with each log.
 * 
 * Example logger call:
 * logger.error('Detailed message', { trace: '0000-cKST', error }); 
 * 
 * Example logger output: 
 * {"trace":"0000-cKST", "error: "Some error message", "level":"error","message":"Detailed message","service":"express-server","timestamp":"2022-04-23 16:08:55"}
 */
export const logger = winston.createLogger({
    // Using `warn` instead of `warning` is a common enough mistake that it's best to support it.
    levels: { ...winston.config.syslog.levels, warn: winston.config.syslog.levels.warning },
    level: getLogLevel(),
    format: winston.format.combine(
        winston.format.errors({ stack: true }),
        winston.format.timestamp({ format: "YYYY-MM-DD HH:mm:ss" }),
        winston.format.json(),
    ),
    defaultMeta: { service: "express-server" },
    transports: getTransports(),
});
