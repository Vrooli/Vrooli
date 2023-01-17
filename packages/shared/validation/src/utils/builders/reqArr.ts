import * as yup from 'yup';
import { req } from './req';

/**
 * Creates an array of a required field
 */
export const reqArr = <T extends yup.AnySchema>(field: T) => {
    return yup.array().of(req(field));
}