import { ApolloError } from 'apollo-server-express';

export class CustomError extends ApolloError {
    constructor(error: any, message?: any) {
        super(message || error.message, error.code);

        Object.defineProperty(this, 'name', { value: 'CustomError' });
    }
}

export async function validateArgs(schema: any, args: any) {
    try {
        await schema.validate(args, { abortEarly: false });
    } catch (err: any) {
        console.info('Invalid arguments')
        return new CustomError({
            code: 'ARGS_VALIDATION_FAILED',
            message: err.errors.toString()
        })
    }
    return null;
}