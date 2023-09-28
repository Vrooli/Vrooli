import * as yup from "yup";

/**
 * Converts a TypeScript enum to a yup oneOf array
 */
export const enumToYup = (enumObj: { [x: string]: any }) =>
    yup.string().trim().removeEmptyString().oneOf(Object.values(enumObj));
