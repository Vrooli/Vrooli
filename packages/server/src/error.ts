import { ApolloError } from 'apollo-server-express';
import { genErrorCode, logger, LogLevel } from './logger';

export class CustomError extends ApolloError {
    constructor(error: any, message?: any, logMeta?: { [key: string]: any }) {
        // Format error
        super(message || error.message, error.code);
        Object.defineProperty(this, 'name', { value: error.code });
        // Log error
        logger.log(LogLevel.error, message ?? error.message, logMeta);
    }
}