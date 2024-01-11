import * as yup from "yup";
import { req } from "./req";

/**
 * Creates an array where the array itself is required,
 * as well as all of the data inside.
 */
export const reqArr = <T extends yup.AnySchema>(field: T) => {
    return req(yup.array().of(field));
};
