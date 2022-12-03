import { ApolloError } from 'apollo-server-express';
import i18next from 'i18next';
import { randomString } from '../auth';
import { ErrorKey } from '../types';
import { logger } from './logger';

/**
 * Generates unique erro code by appending 
 * a unique string with a randomly generated string. 
 * This way, you can locate both the location in the code which 
 * generated the error, and the exact line in the log file where 
 * the error occurred.
 * @param locationCode String representing the location in the code where the error occurred, ideally 4 characters long.
 * @returns `${locationCode}-${randomString}`, where the random string is 4 characters long.
 */
function genTrace(locationCode: string): string {
    return `${locationCode}-${randomString(4)}`;
}

export class CustomError extends ApolloError {
    constructor(traceBase: string, errorCode: ErrorKey, languages: string[], data?: { [key: string]: any }) {
        // Find message in user's language
        const lng = languages.length > 0 ? languages[0] : 'en';
        const message = i18next.t(`error:${errorCode}`, { lng }) ?? errorCode
        // Generate unique trace
        const trace = genTrace(traceBase);
        // Format error
        super(message, errorCode);
        Object.defineProperty(this, 'name', { value: errorCode });
        // Log error, if trace is provided
        if (trace) {
            const { msg, trace, ...rest } = data ?? {};
            logger.error({ msg: message, trace, ...rest });
        }
    }
}