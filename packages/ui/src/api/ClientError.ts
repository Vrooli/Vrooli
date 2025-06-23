/**
 * Client-Side Error Implementation
 * 
 * This class provides a client-side implementation of the VrooliError interface
 * that integrates seamlessly with ServerResponseParser for UI error handling.
 */

import { type ServerError, type TranslationKeyError, type ParseableError } from "@vrooli/shared";

/**
 * Client-side error implementation that integrates with ServerResponseParser.
 * This allows UI components to work with errors in a consistent way across
 * the entire application stack.
 */
export class ClientError extends Error implements ParseableError {
    public code: TranslationKeyError;
    public trace: string;
    public data?: Record<string, any>;

    constructor(code: TranslationKeyError, trace: string, data?: Record<string, any>) {
        super(`${code}: ${trace}`);
        this.code = code;
        this.trace = trace;
        this.data = data;
        Object.defineProperty(this, "name", { value: code });
    }

    /**
     * Create ClientError from ServerError (what ServerResponseParser works with).
     * This is the primary way to convert server errors to client errors.
     */
    static fromServerError(serverError: ServerError): ClientError {
        if ("code" in serverError) {
            // Translated error with error code
            return new ClientError(serverError.code, serverError.trace);
        } else if ("message" in serverError) {
            // Untranslated error with direct message - use a proper translation key
            return new ClientError("ErrorUnknown" as TranslationKeyError, serverError.trace, {
                originalMessage: serverError.message,
            });
        }
        throw new Error("Invalid ServerError format - must have either 'code' or 'message'");
    }

    /**
     * Create multiple ClientErrors from ServerResponse errors array.
     */
    static fromServerErrors(serverErrors: ServerError[]): ClientError[] {
        return serverErrors.map(error => this.fromServerError(error));
    }

    /**
     * Convert back to ServerError format for API compatibility.
     */
    toServerError(): ServerError {
        // If this was created from an untranslated error, preserve the original message
        if (this.data?.originalMessage) {
            return { 
                trace: this.trace, 
                message: this.data.originalMessage as string, 
            };
        }
        return { trace: this.trace, code: this.code };
    }

    /**
     * Check if this error can be parsed by ServerResponseParser.
     * ClientErrors are designed to be parseable by default.
     */
    isParseableByUI(): boolean {
        return true;
    }

    /**
     * Get severity level for UI display.
     * This can be enhanced based on error code patterns.
     */
    getSeverity(): "Error" | "Warning" | "Info" {
        const codeStr = this.code.toLowerCase();
        
        if (codeStr.includes("warning") || codeStr.includes("warn")) {
            return "Warning";
        }
        
        if (codeStr.includes("info") || codeStr.includes("notice")) {
            return "Info";
        }
        
        return "Error";
    }

    /**
     * Get user-friendly message for fallback scenarios.
     * This is used when i18n translation is not available.
     */
    getUserMessage(languages: string[] = ["en"]): string {
        // If we have an original message from untranslated error, use it
        if (this.data?.originalMessage) {
            return this.data.originalMessage as string;
        }
        
        // For translated errors, return the code as fallback
        // (in practice, ServerResponseParser would handle i18n translation)
        return `Error: ${this.code}`;
    }

    /**
     * Check if this error represents an untranslated server error.
     */
    isUntranslated(): boolean {
        return this.code === "ErrorUnknown" && !!this.data?.originalMessage;
    }

    /**
     * Get the original untranslated message if available.
     */
    getOriginalMessage(): string | undefined {
        return this.data?.originalMessage as string | undefined;
    }

    /**
     * Validate that this ClientError is compatible with ServerResponseParser.
     * This replaces the need for external validation utilities.
     */
    isCompatibleWithParser(): boolean {
        try {
            const serverError = this.toServerError();
            return serverError && 
                   (("code" in serverError && typeof serverError.code === "string") ||
                    ("message" in serverError && typeof serverError.message === "string")) &&
                   typeof serverError.trace === "string";
        } catch {
            return false;
        }
    }

    /**
     * Validate that this ClientError follows the VrooliError interface pattern.
     */
    matchesVrooliErrorPattern(): boolean {
        return typeof this.code === "string" && 
               typeof this.trace === "string" &&
               typeof this.toServerError === "function";
    }
}