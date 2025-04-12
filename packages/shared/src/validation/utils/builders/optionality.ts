import * as yup from "yup";
import { reqErr } from "../errors.js";

/**
 * Appends .notRequired().default(undefined) to a yup field
 * When a field is optional, it should not validate its children if it's undefined
 */
export function opt<T extends yup.AnySchema>(field: T) {
    return field.notRequired().nullable().default(undefined);
}

/**
 * Creates an array where the array itself is optional,
 * but if provided, all of the data inside should be required.
 * Don't strip the array when validating
 */
export function optArr<T extends yup.AnySchema>(field: T) {
    return yup.array().of(field).notRequired().nullable();
}

/**
 * Appends .required(reqErr) to a yup field
 */
export function req<T extends yup.AnySchema>(field: T) {
    return field.required(reqErr);
}

/**
 * Creates an array where the array itself is required,
 * as well as all of the data inside.
 */
export function reqArr<T extends yup.AnySchema>(field: T) {
    return req(yup.array().of(field));
}
