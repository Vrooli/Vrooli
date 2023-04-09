import { MessageParams } from 'yup/lib/types';

/**
 * Error message for min string length
 */
export const minStrErr = (params: { min: number } & MessageParams) => {
    const amountUnder = params.min - params.value.length;
    if (amountUnder === 1) {
        return "1 character under the limit";
    } else {
        return `${amountUnder} characters under the limit`;
    }
}