import ora, { type Ora } from "ora";
import { handleCommandError } from "./errorHandler.js";

export interface SpinnerOptions {
    /** The message to display */
    text: string;
    /** Whether to show the spinner (false for JSON output mode) */
    enabled?: boolean;
    /** Success message (defaults to original text) */
    successText?: string;
    /** Fail message (defaults to original text) */
    failText?: string;
}

/**
 * Execute an async operation with a spinner
 * Automatically handles success/failure states
 */
export async function withSpinner<T>(
    options: SpinnerOptions | string,
    action: () => Promise<T>,
): Promise<T> {
    const config: SpinnerOptions = typeof options === "string" 
        ? { text: options }
        : options;

    const enabled = config.enabled !== false && !process.env.JSON_OUTPUT;
    
    if (!enabled) {
        // Just run the action without spinner in JSON mode
        return action();
    }

    const spinner = ora(config.text).start();

    try {
        const result = await action();
        spinner.succeed(config.successText || config.text);
        return result;
    } catch (error) {
        spinner.fail(config.failText || config.text);
        throw error;
    }
}

/**
 * Create a reusable spinner instance for multi-step operations
 */
export class SpinnerManager {
    private spinner: Ora | null = null;
    private enabled: boolean;

    constructor(enabled = true) {
        this.enabled = enabled && !process.env.JSON_OUTPUT;
    }

    /**
     * Start a new spinner or update existing one
     */
    start(text: string): void {
        if (!this.enabled) return;

        if (this.spinner) {
            this.spinner.text = text;
        } else {
            this.spinner = ora(text).start();
        }
    }

    /**
     * Update spinner text
     */
    update(text: string): void {
        if (!this.enabled || !this.spinner) return;
        this.spinner.text = text;
    }

    /**
     * Mark current step as successful and start next
     */
    succeed(text?: string): void {
        if (!this.enabled || !this.spinner) return;
        this.spinner.succeed(text);
        this.spinner = null;
    }

    /**
     * Mark current step as failed
     */
    fail(text?: string): void {
        if (!this.enabled || !this.spinner) return;
        this.spinner.fail(text);
        this.spinner = null;
    }

    /**
     * Mark current step as warning
     */
    warn(text?: string): void {
        if (!this.enabled || !this.spinner) return;
        this.spinner.warn(text);
        this.spinner = null;
    }

    /**
     * Stop spinner without status
     */
    stop(): void {
        if (!this.enabled || !this.spinner) return;
        this.spinner.stop();
        this.spinner = null;
    }

    /**
     * Info message (spinner with info symbol)
     */
    info(text: string): void {
        if (!this.enabled || !this.spinner) return;
        this.spinner.info(text);
        this.spinner = null;
    }
}

/**
 * Execute multiple steps with progress indication
 */
export async function withProgress(
    steps: Array<{
        text: string;
        action: () => Promise<unknown>;
        successText?: string;
    }>,
): Promise<void> {
    const spinner = new SpinnerManager();

    for (const [index, step] of steps.entries()) {
        const stepText = `[${index + 1}/${steps.length}] ${step.text}`;
        spinner.start(stepText);

        try {
            await step.action();
            spinner.succeed(step.successText || stepText);
        } catch (error) {
            spinner.fail(stepText);
            handleCommandError(error, undefined, step.text);
        }
    }
}
