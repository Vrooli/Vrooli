import { type TranslationKeyError } from "@local/shared";
import { randomString } from "../auth/codes";
import { logger } from "./logger";

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
    /** The base trace (unique identifier) for the error */
    public traceBase: string;

    constructor(traceBase: string, errorCode: TranslationKeyError, data?: ErrorTrace) {
        // Generate unique trace
        const trace = genTrace(traceBase);
        super(`${errorCode}: ${trace}`);
        this.code = errorCode;
        this.traceBase = traceBase;
        Object.defineProperty(this, "name", { value: errorCode });
        logger.error({ ...(data ?? {}), msg: errorCode, trace });
    }
}
