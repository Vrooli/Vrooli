import { MessageParams } from "yup/lib/types";

/**
 * Error message for min number
 */
export const minNumErr = (params: { min: number } & MessageParams) => {
    return `Minimum value is ${params.min}`;
};
