import { type Command } from "commander";
import ora, { type Ora } from "ora";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";

export interface CommandOptions {
    json?: boolean;
    profile?: string;
    verbose?: boolean;
}

export abstract class BaseCommand {
    protected constructor(
        protected program: Command,
        protected client: ApiClient,
        protected config: ConfigManager,
    ) {
        this.registerCommands();
    }

    /**
     * Register all commands for this command group
     */
    protected abstract registerCommands(): void;

    /**
     * Execute an operation with a spinner
     */
    protected async executeWithSpinner<T>(
        message: string,
        operation: () => Promise<T>,
        successMessage?: string,
    ): Promise<T> {
        const spinner = this.createSpinner(message);
        
        try {
            spinner.start();
            const result = await operation();
            spinner.succeed(successMessage || message);
            return result;
        } catch (error) {
            spinner.fail(`Failed: ${message}`);
            throw error;
        }
    }

    /**
     * Create a spinner (abstracted for testing)
     */
    protected createSpinner(message: string): Ora {
        return ora(message);
    }

    /**
     * Handle errors consistently across all commands
     */
    protected handleError(error: unknown, context?: string): never {
        const errorMessage = error instanceof Error ? error.message : String(error);
        const formattedMessage = context ? `${context}: ${errorMessage}` : errorMessage;
        
        // Always call output first (needed for tests that verify output calls)
        if (this.config.isJsonOutput()) {
            output.json({ 
                error: true, 
                message: errorMessage,
                context,
            });
        } else {
            output.error(formattedMessage);
        }
        
        // In test environments, throw the error to allow proper test assertions
        // Check for NODE_ENV=test (set in test-setup.ts)
        if (process.env.NODE_ENV === "test") {
            throw new Error(formattedMessage);
        }
        
        process.exit(1);
    }

    /**
     * Output data in the appropriate format (JSON or normal)
     */
    protected output(data: unknown, defaultFormatter?: () => void): void {
        if (this.config.isJsonOutput()) {
            output.json(data);
        } else if (defaultFormatter) {
            defaultFormatter();
        } else {
            output.info(String(data));
        }
    }

    /**
     * Display data in a table format
     */
    protected displayTable(data: Array<Record<string, unknown>>): void {
        if (this.config.isJsonOutput()) {
            output.json(data);
        } else {
            output.table(data);
        }
    }

    /**
     * Check if the user is authenticated
     */
    protected requireAuth(): void {
        const token = this.config.getAuthToken();
        if (!token) {
            this.handleError(
                new Error("You must be logged in to use this command"),
                "Authentication required",
            );
        }
    }

    /**
     * Parse and validate a JSON file
     */
    protected async parseJsonFile<T>(filePath: string): Promise<T> {
        const fs = await import("fs/promises");
        try {
            const content = await fs.readFile(filePath, "utf-8");
            return JSON.parse(content) as T;
        } catch (error) {
            if ((error as NodeJS.ErrnoException).code === "ENOENT") {
                throw new Error(`File not found: ${filePath}`);
            }
            if (error instanceof SyntaxError) {
                throw new Error(`Invalid JSON in file: ${filePath}`);
            }
            throw error;
        }
    }

    /**
     * Write data to a JSON file
     */
    protected async writeJsonFile(filePath: string, data: unknown): Promise<void> {
        const fs = await import("fs/promises");
        const path = await import("path");
        
        // Ensure directory exists
        const dir = path.dirname(filePath);
        await fs.mkdir(dir, { recursive: true });
        
        // Write file
        await fs.writeFile(filePath, JSON.stringify(data, null, 2));
    }

    /**
     * Common validation methods
     */
    protected validateEmail(email: string): boolean {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }

    protected validateUuid(uuid: string): boolean {
        const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
        return uuidRegex.test(uuid);
    }

    /**
     * Log debug information if verbose mode is enabled
     */
    protected debug(message: string, data?: unknown): void {
        if (this.config.isDebug()) {
            if (data) {
                console.debug(`[DEBUG] ${message}:`, data);
            } else {
                console.debug(`[DEBUG] ${message}`);
            }
        }
    }
}
