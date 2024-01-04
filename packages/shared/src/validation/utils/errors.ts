import { MessageParams } from "yup/lib/types";

/** Error message for max number */
export const maxNumErr = (params: { max: number }) => {
    return `Minimum value is ${params.max}`;
};

/** Error message for max string length */
export const maxStrErr = (params: { max: number } & MessageParams) => {
    const amountOver = params.value.length - params.max;
    if (amountOver === 1) {
        return "1 character over the limit";
    } else {
        return `${amountOver} characters over the limit`;
    }
};

/** Error message for min number */
export const minNumErr = (params: { min: number } & MessageParams) => {
    return `Minimum value is ${params.min}`;
};

/** Error message for min string length */
export const minStrErr = (params: { min: number } & MessageParams) => {
    const amountUnder = params.min - params.value.length;
    if (amountUnder === 1) {
        return "1 character under the limit";
    } else {
        return `${amountUnder} characters under the limit`;
    }
};

/** Error message for required field */
export const reqErr = () => "This field is required";
