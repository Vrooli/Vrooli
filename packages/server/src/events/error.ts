import { ServerError, type TranslationKeyError } from "@local/shared";
import { randomString } from "../auth/codes.js";
import { logger } from "./logger.js";

const TRACE_LENGTH = 4;

export type ErrorTrace = Record<string, any>;

/**
 * Generates unique error code by appending 
 * a unique string with a randomly generated string. 
 * This way, you can locate both the location in the code which 
 * generated the error, and the exact line in the log file where 
 * the error occurred.
 * @param locationCode String representing the location in the code where the error occurred, ideally 4 characters long.
 * @returns `${locationCode}-${randomString}`, where the random string is 4 characters long.
 */
function genTrace(locationCode: string): string {
    return `${locationCode}-${randomString(TRACE_LENGTH)}`;
}

export class CustomError extends Error {
    /** The translation key for the error message */
    public code: TranslationKeyError;
    /** The full trace (unique identifier for spot in codebase + random string to locate in logs) */
    public trace: string;

    constructor(traceBase: string, errorCode: TranslationKeyError, data?: ErrorTrace) {
        // Generate unique trace
        const trace = genTrace(traceBase);
        super(`${errorCode}: ${trace}`);
        this.code = errorCode;
        this.trace = trace;
        Object.defineProperty(this, "name", { value: errorCode });
        logger.error({ ...(data ?? {}), msg: errorCode, trace });
    }

    toServerError(): ServerError {
        return { trace: this.trace, code: this.code };
    }
}

/**
 * Creates a standard error instance for invalid user credentials or authentication failures.
 * 
 * Intentionally vague for security purposes during login/password reset flows.
 * Generates a unique trace each time it's called.
 * 
 * @returns A new CustomError instance with code "InvalidCredentials" and trace "0062-xxxx".
 */
export function createInvalidCredentialsError(): CustomError {
    // Pass the base trace code "0062" and the error code "InvalidCredentials"
    return new CustomError("0062", "InvalidCredentials");
}
