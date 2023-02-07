/**
 * Preferred logging method for dev logs. Allows you to specify 
 * the level and output location(s). Includes timestamp 
 * with each log.
 * 
 * Example logger call:
 * logger.error('Detailed message', { trace: '0000-cKST', error }); 
 * 
 * Example logger output: 
 * {"trace":"0000-cKST", "error: "Some error message", "level":"error","message":"Detailed message","service":"express-server","timestamp":"2022-04-23 16:08:55"}
 */
import winston from 'winston';

const LOG_DIR = `${process.env.PROJECT_DIR}/data/logs`;

export const logger = winston.createLogger({
    levels: winston.config.syslog.levels,
    format: winston.format.combine(
        winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
        winston.format.json()
    ),
    defaultMeta: { service: 'express-server' },
    transports: [
        // Errors are not only included in the combined log file, but also in their own file
        new winston.transports.File({ 
            filename: `${LOG_DIR}/error.log`, 
            level: 'error',
            maxsize: 5242880, // 5MB
        }),
        new winston.transports.File({ 
            filename: `${LOG_DIR}/combined.log`,
            maxsize: 5242880, // 5MB
        }),
    ],
});

/**
 * Logs to the console if not in production. 
 * Format: `${info.level}: ${info.message} JSON.stringify({ ...rest }) `.
 * Be careful not to add any data with circular references, as this will break JSON.stringify.
 */
if (process.env.NODE_ENV === 'development') {
    logger.add(new winston.transports.Console({
        format: winston.format.simple(),
    }));
}