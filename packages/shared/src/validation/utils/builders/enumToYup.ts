import * as yup from "yup";

/**
 * Converts a TypeScript enum to a yup oneOf array
 */
export function enumToYup(enumObj: { [x: string]: any }) {
    return yup.string().trim().removeEmptyString().oneOf(Object.values(enumObj));
}
