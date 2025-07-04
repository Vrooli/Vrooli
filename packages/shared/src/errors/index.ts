/**
 * Shared Error Architecture
 * 
 * This module provides the core error interfaces and utilities that ensure
 * consistency between server CustomError and client error handling.
 * 
 * The architecture follows this flow:
 * Server: CustomError → ServerError → API Transport → UI: ServerResponseParser → ClientError
 */

import { type TranslationKeyError } from "../types.d.js";
import { type ServerError } from "../consts/api.js";

/**
 * Core error interface that all error implementations must follow.
 * This ensures consistency between server CustomError and client error handling.
 */
export interface VrooliError {
    /** The translation key for the error message */
    code: TranslationKeyError;
    /** The full trace (unique identifier + random string) */
    trace: string;
    /** Convert to server-safe format for API responses */
    toServerError(): ServerError;
    /** Optional error data/context */
    data?: Record<string, any>;
}

/**
 * Enhanced error interface that includes UI parsing capabilities.
 * This bridges server errors and UI error handling via ServerResponseParser.
 */
export interface ParseableError extends VrooliError {
    /** Check if this error can be parsed by ServerResponseParser */
    isParseableByUI(): boolean;
    /** Get user-friendly message (for fallback scenarios) */
    getUserMessage?(languages?: string[]): string;
    /** Get severity level for UI display */
    getSeverity?(): "Error" | "Warning" | "Info";
}

/**
 * Validation utilities for error objects.
 * Used by error fixtures and tests to ensure consistency.
 */
export interface ErrorValidator {
    /** Validate trace format (should be XXXX-XXXX pattern) */
    isValidTrace(trace: string): boolean;
    /** Validate error code format */
    isValidErrorCode(code: string): boolean;
    /** Check if object matches VrooliError interface */
    matchesPattern(error: any): error is VrooliError;
    /** Check if error is compatible with ServerResponseParser */
    isCompatibleWithParser(error: any): boolean;
}

/**
 * Standard error validator implementation.
 * Used throughout the codebase for consistent error validation.
 */
export const standardErrorValidator: ErrorValidator = {
    isValidTrace: (trace: string): boolean => {
        // Matches server pattern: 4 digits, dash, 4 alphanumeric characters
        return /^\d{4}-\w{4}$/.test(trace);
    },
    
    isValidErrorCode: (code: string): boolean => {
        return typeof code === "string" && code.length > 0;
    },
    
    matchesPattern: (error: any): error is VrooliError => {
        return error && 
               typeof error.code === "string" && 
               typeof error.trace === "string" &&
               typeof error.toServerError === "function";
    },

    isCompatibleWithParser: (error: any): boolean => {
        try {
            const serverError = error?.toServerError?.();
            return serverError && 
                   (("code" in serverError && typeof serverError.code === "string") ||
                    ("message" in serverError && typeof serverError.message === "string")) &&
                   typeof serverError.trace === "string";
        } catch {
            return false;
        }
    },
};

/**
 * Factory function type for creating errors.
 * Allows different implementations while maintaining interface consistency.
 */
export type ErrorFactory = (
    traceBase: string, 
    errorCode: TranslationKeyError, 
    data?: Record<string, any>
) => VrooliError;
