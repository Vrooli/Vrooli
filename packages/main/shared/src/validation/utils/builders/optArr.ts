import * as yup from "yup";
import { opt } from "./opt";


/**
 * Creates an array of an optional field
 */
export const optArr = <T extends yup.AnySchema>(field: T) => {
    return yup.array().of(opt(field));
};
