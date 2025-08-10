import { logger } from "./logger.js";
import chalk from "chalk";

/**
 * Standardized output utility for CLI commands
 * Provides consistent formatting and output methods
 */
export const output = {
    /**
     * Display informational message
     */
    info: (message: string, ...args: unknown[]): void => {
        logger.info(message, ...args);
    },

    /**
     * Display success message (without prefix for cleaner output)
     */
    success: (message: string, ...args: unknown[]): void => {
        console.log(chalk.green(message), ...args);
    },

    /**
     * Display error message
     */
    error: (message: string, error?: Error | unknown): void => {
        if (error instanceof Error) {
            logger.error(message, error.message);
            if (process.env.DEBUG === "true") {
                console.error(error.stack);
            }
        } else if (error) {
            logger.error(message, error);
        } else {
            logger.error(message);
        }
    },

    /**
     * Display warning message
     */
    warn: (message: string, ...args: unknown[]): void => {
        logger.warn(message, ...args);
    },

    /**
     * Display debug message (only in debug mode)
     */
    debug: (message: string, ...args: unknown[]): void => {
        logger.debug(message, ...args);
    },

    /**
     * Display data in table format
     */
    table: (data: Record<string, unknown>[]): void => {
        console.table(data);
    },

    /**
     * Display raw output (for JSON mode or when formatting should be preserved)
     */
    raw: (data: unknown): void => {
        console.log(data);
    },

    /**
     * Display JSON output
     */
    json: (data: unknown): void => {
        console.log(JSON.stringify(data, null, 2));
    },

    /**
     * Display a blank line
     */
    newline: (): void => {
        console.log();
    },

    /**
     * Display a section header
     */
    section: (title: string): void => {
        console.log();
        console.log(chalk.bold.underline(title));
        console.log();
    },

    /**
     * Display a list item
     */
    listItem: (item: string, indent = 0): void => {
        const prefix = "  ".repeat(indent) + "• ";
        console.log(prefix + item);
    },

    /**
     * Display key-value pair
     */
    keyValue: (key: string, value: unknown, indent = 0): void => {
        const prefix = "  ".repeat(indent);
        console.log(prefix + chalk.bold(key + ":") + " " + String(value));
    },
};
