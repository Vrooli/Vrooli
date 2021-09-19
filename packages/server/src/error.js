import { ApolloError } from 'apollo-server-express';

export class CustomError extends ApolloError {
    constructor(error) {
        super(error.message, error.code);

        Object.defineProperty(this, 'name', { value: 'CustomError' });
    }
}

export async function validateArgs(schema, args) {
    try {
        await schema.validate(args, { abortEarly: false });
    } catch (err) {
        console.info('Invalid arguments')
        return new CustomError({
            code: 'ARGS_VALIDATION_FAILED',
            message: err.errors
        })
    }
    return null;
}