interface MessageParams {
    value: unknown;
}

/** Error message for max number */
export function maxNumErr(params: { max: number }) {
    return `Maximum value is ${params.max}`;
}

/** Error message for max string length */
export function maxStrErr(params: { max: number } & MessageParams) {
    if (typeof params.value !== "string") throw new Error("Value must be a string");
    const amountOver = params.value.length - params.max;
    if (amountOver === 1) {
        return "1 character over the limit";
    } else {
        return `${amountOver} characters over the limit`;
    }
}

/** Error message for min number */
export function minNumErr(params: { min: number }) {
    return `Minimum value is ${params.min}`;
}

/** Error message for min string length */
export function minStrErr(params: { min: number } & MessageParams) {
    if (typeof params.value !== "string") throw new Error("Value must be a string");
    const amountUnder = params.min - params.value.length;
    if (amountUnder === 1) {
        return "1 character under the limit";
    } else {
        return `${amountUnder} characters under the limit`;
    }
}

/** Error message for required field */
export function reqErr() {
    return "This field is required";
}
