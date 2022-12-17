import { MessageParams } from 'yup/lib/types';

/**
 * Error message for max string length
 */
export const maxStrErr = (params: { max: number } & MessageParams) => {
    const amountOver = params.value.length - params.max;
    if (amountOver === 1) {
        return "1 character over the limit";
    } else {
        return `${amountOver} characters over the limit`;
    }
}