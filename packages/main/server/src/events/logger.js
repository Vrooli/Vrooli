import winston from "winston";
const LOG_DIR = `${process.env.PROJECT_DIR}/data/logs`;
export const logger = winston.createLogger({
    levels: winston.config.syslog.levels,
    format: winston.format.combine(winston.format.timestamp({ format: "YYYY-MM-DD HH:mm:ss" }), winston.format.json()),
    defaultMeta: { service: "express-server" },
    transports: [
        new winston.transports.File({
            filename: `${LOG_DIR}/error.log`,
            level: "error",
            maxsize: 5242880,
        }),
        new winston.transports.File({
            filename: `${LOG_DIR}/combined.log`,
            maxsize: 5242880,
        }),
    ],
});
if (process.env.NODE_ENV === "development") {
    logger.add(new winston.transports.Console({
        format: winston.format.simple(),
    }));
}
//# sourceMappingURL=logger.js.map