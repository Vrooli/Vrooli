import * as yup from "yup";
import { opt } from "./opt";

/**
 * Creates an array where the array itself is optional,
 * but if provided, all of the data inside should be required.
 */
export const optArr = <T extends yup.AnySchema>(field: T) => {
    return opt(yup.array().of(field));
};
