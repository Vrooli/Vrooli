import * as yup from 'yup';

/**
 * Appends .notRequired().default(undefined) to a yup field
 */
export const opt = <T extends yup.AnySchema>(field: T) => {
    return field.notRequired().default(undefined);
}