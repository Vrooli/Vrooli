import chalk from "chalk";

// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-14

export interface Logger {
    debug: (message: string, ...args: unknown[]) => void;
    info: (message: string, ...args: unknown[]) => void;
    success: (message: string, ...args: unknown[]) => void;
    warn: (message: string, ...args: unknown[]) => void;
    error: (message: string, ...args: unknown[]) => void;
    setLevel: (level: string) => void;
}

class SimpleLogger implements Logger {
    private level = "info";
    private levels: Record<string, number> = {
        debug: 0,
        info: 1,
        warn: 2,
        error: 3,
    };

    setLevel(level: string): void {
        this.level = level;
    }

    private shouldLog(level: string): boolean {
        return this.levels[level] >= this.levels[this.level];
    }

    debug(message: string, ...args: unknown[]): void {
        if (this.shouldLog("debug")) {
            console.log(chalk.gray(`[DEBUG] ${message}`), ...args);
        }
    }

    info(message: string, ...args: unknown[]): void {
        if (this.shouldLog("info")) {
            console.log(chalk.blue(`[INFO] ${message}`), ...args);
        }
    }

    success(message: string, ...args: unknown[]): void {
        if (this.shouldLog("info")) {
            console.log(chalk.green(`✓ ${message}`), ...args);
        }
    }

    warn(message: string, ...args: unknown[]): void {
        if (this.shouldLog("warn")) {
            console.warn(chalk.yellow(`[WARN] ${message}`), ...args);
        }
    }

    error(message: string, ...args: unknown[]): void {
        if (this.shouldLog("error")) {
            console.error(chalk.red(`[ERROR] ${message}`), ...args);
        }
    }
}

export const logger = new SimpleLogger();
