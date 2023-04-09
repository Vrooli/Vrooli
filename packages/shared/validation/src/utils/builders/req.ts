import * as yup from 'yup';
import { reqErr } from '../errors';

/**
 * Appends .required(reqErr) to a yup field
 */
export const req = <T extends yup.AnySchema>(field: T) => {
    return field.required(reqErr);
}