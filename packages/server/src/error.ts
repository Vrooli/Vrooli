import { ApolloError } from 'apollo-server-express';

export class CustomError extends ApolloError {
    constructor(error: any, message?: any) {
        super(message || error.message, error.code);

        Object.defineProperty(this, 'name', { value: error.code });
    }
}