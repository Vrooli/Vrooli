/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import * as yup from 'yup';
import { MessageParams } from 'yup/lib/types';

/**
 * Error message for max string length
 */
export const maxStringErrorMessage = (params: { max: number } & MessageParams) => {
    const amountOver = params.value.length - params.max;
    if (amountOver === 1) {
        return "1 character over the limit";
    } else {
        return `${amountOver} characters over the limit`;
    }
}

/**
 * Error message for min string length
 */
export const minStringErrorMessage = (params: { min: number } & MessageParams) => {
    const amountUnder = params.min - params.value.length;
    if (amountUnder === 1) {
        return "1 character under the limit";
    } else {
        return `${amountUnder} characters under the limit`;
    }
}

/**
 * Error message for max number
 */
export const maxNumberErrorMessage = (params: { max: number } & MessageParams) => {
    return `Minimum value is ${params.max}`;
}

/**
 * Error message for min number
 */
export const minNumberErrorMessage = (params: { min: number } & MessageParams) => {
    return `Minimum value is ${params.min}`;
}

/**
 * Error message for required field
 */
export const requiredErrorMessage = (params: MessageParams) => {
    return `This field is required`;
}

export const id = yup.string().max(256, maxStringErrorMessage)
export const bio = yup.string().max(2048, maxStringErrorMessage)
export const description = yup.string().max(2048, maxStringErrorMessage)
export const helpText = yup.string().max(2048, maxStringErrorMessage)
export const language = yup.string().min(2, minStringErrorMessage).max(3, maxStringErrorMessage) // Language code
export const name = yup.string().min(3, minStringErrorMessage).max(128, maxStringErrorMessage)
export const handle = yup.string().min(3, minStringErrorMessage).max(16, maxStringErrorMessage).nullable() // ADA Handle
export const tag = yup.string().min(2, minStringErrorMessage).max(64, maxStringErrorMessage)
export const title = yup.string().min(2, minStringErrorMessage).max(128, maxStringErrorMessage)
export const version = yup.string().max(16, maxStringErrorMessage)
export const idArray = yup.array().of(id.required(requiredErrorMessage))
export const tagArray = yup.array().of(tag.required(requiredErrorMessage))