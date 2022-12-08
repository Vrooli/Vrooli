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

/**
 * Calculates major, moderate, and minor versions from a version string
 * Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
 * Ex: 1 => major = 1, moderate = 0, minor = 0
 * Ex: 1.2 => major = 1, moderate = 2, minor = 0
 * Ex: asdfasdf (or any other invalid number) => major = 1, moderate = 0, minor = 0
 * @param version Version string
 * @returns Major, moderate, and minor versions
 */
export const calculateVersionsFromString = (version: string): { major: number, moderate: number, minor: number } => {
    const [major, moderate, minor] = version.split('.').map(v => parseInt(v));
    return {
        major: major || 0,
        moderate: moderate || 0,
        minor: minor || 0
    }
}

/**
 * Determines if a version is greater than or equal to the minimum version
 * @param version Version string
 * @param minimumVersion Minimum version string
 * @returns True if version is greater than or equal to minimum version
 */
export const meetsMinimumVersion = (version: string, minimumVersion: string): boolean => {
    // Parse versions
    const { major: major1, moderate: moderate1, minor: minor1 } = calculateVersionsFromString(version);
    const { major: major2, moderate: moderate2, minor: minor2 } = calculateVersionsFromString(minimumVersion);
    // Return false if major version is less than minimum
    if (major1 < major2) return false;
    // Return false if major version is equal to minimum and moderate version is less than minimum
    if (major1 === major2 && moderate1 < moderate2) return false;
    // Return false if major and moderate versions are equal to minimum and minor version is less than minimum
    if (major1 === major2 && moderate1 === moderate2 && minor1 < minor2) return false;
    // Otherwise, return true
    return true;
}

/**
 * Test for minimum version
 */
export const minVersionTest = (minVersion: string): [string, string, (value: string | undefined) => boolean] => {
    return [
        'version',
        `Minimum version is ${minVersion}`,
        (value: string | undefined) => {
            if (!value) return true;
            return meetsMinimumVersion(value, minVersion);
        }
    ]
}

/**
 * Transform for converting empty/whitespace strings to undefined
 */
export const blankToUndefined = (value: string | undefined) => {
    if (!value) return undefined;
    const trimmed = value.trim();
    if (trimmed === '') return undefined;
    return trimmed;
}

export const id = yup.string().transform(blankToUndefined).max(256, maxStringErrorMessage)
export const bio = yup.string().transform(blankToUndefined).max(2048, maxStringErrorMessage);
export const email = yup.string().transform(blankToUndefined).email().max(256, maxStringErrorMessage);
export const description = yup.string().transform(blankToUndefined).max(2048, maxStringErrorMessage)
export const helpText = yup.string().transform(blankToUndefined).max(2048, maxStringErrorMessage)
export const language = yup.string().transform(blankToUndefined).min(2, minStringErrorMessage).max(3, maxStringErrorMessage) // Language code
export const name = yup.string().transform(blankToUndefined).min(3, minStringErrorMessage).max(128, maxStringErrorMessage)
export const handle = yup.string().transform(blankToUndefined).min(3, minStringErrorMessage).max(16, maxStringErrorMessage).nullable() // ADA Handle
export const tag = yup.string().transform(blankToUndefined).min(2, minStringErrorMessage).max(64, maxStringErrorMessage)
export const version = (minVersion: string = '0.0.1') => yup.string().transform(blankToUndefined).max(16, maxStringErrorMessage).test(...minVersionTest(minVersion))
export const idArray = yup.array().of(id.required(requiredErrorMessage))
export const tagArray = yup.array().of(tag.required(requiredErrorMessage))