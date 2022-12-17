import { MessageParams } from 'yup/lib/types';

/**
 * Error message for max number
 */
export const maxNumErr = (params: { max: number } & MessageParams) => {
    return `Minimum value is ${params.max}`;
}