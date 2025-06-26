import chalk from "chalk";

export interface Logger {
    debug: (message: string, ...args: any[]) => void;
    info: (message: string, ...args: any[]) => void;
    warn: (message: string, ...args: any[]) => void;
    error: (message: string, ...args: any[]) => void;
    setLevel: (level: string) => void;
}

class SimpleLogger implements Logger {
    private level: string = "info";
    private levels: Record<string, number> = {
        debug: 0,
        info: 1,
        warn: 2,
        error: 3
    };

    setLevel(level: string): void {
        this.level = level;
    }

    private shouldLog(level: string): boolean {
        return this.levels[level] >= this.levels[this.level];
    }

    debug(message: string, ...args: any[]): void {
        if (this.shouldLog("debug")) {
            console.log(chalk.gray(`[DEBUG] ${message}`), ...args);
        }
    }

    info(message: string, ...args: any[]): void {
        if (this.shouldLog("info")) {
            console.log(chalk.blue(`[INFO] ${message}`), ...args);
        }
    }

    warn(message: string, ...args: any[]): void {
        if (this.shouldLog("warn")) {
            console.warn(chalk.yellow(`[WARN] ${message}`), ...args);
        }
    }

    error(message: string, ...args: any[]): void {
        if (this.shouldLog("error")) {
            console.error(chalk.red(`[ERROR] ${message}`), ...args);
        }
    }
}

export const logger = new SimpleLogger();