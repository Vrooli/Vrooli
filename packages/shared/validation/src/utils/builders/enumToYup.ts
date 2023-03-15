import * as yup from 'yup';
import { blankToUndefined } from './blankToUndefined';

/**
 * Converts a TypeScript enum to a yup oneOf array
 */
export const enumToYup = (enumObj: { [x: string]: any }) =>
    yup.string().transform(blankToUndefined).oneOf(Object.values(enumObj));
