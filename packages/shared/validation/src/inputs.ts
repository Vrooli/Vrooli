/**
 * Validation schemas for all standards/inputs/outputs
 */
import { requiredErrorMessage } from 'base';
import * as yup from 'yup';

const required = yup.boolean();

export const inputYup = yup.object().shape({
    required: required.notRequired().default(undefined),
    type: yup.string().oneOf(['string']).default('string'),
    // checks: 
})

export const textFieldStandardInputForm = yup.object().shape({
    defaultValue: yup.string().notRequired().default(undefined),
    autoComplete: yup.string().notRequired().default(undefined),
    maxRows: yup.number().notRequired().default(undefined),
    yup: inputYup.notRequired().default(undefined),
})

export const jsonStandardInputForm = yup.object().shape({
    format: yup.string().required(requiredErrorMessage),
    defaultValue: yup.string().notRequired().default(undefined),
    // Object with keys of the format: { label?: string, helperText?: string, yup?: inputYup, defaultValue?: string | object }
    variables: yup.object().test(
        'variables',
        'Variables must be an object with keys of the format { label?: string, helperText?: string, yup?: inputYup, defaultValue?: string | object }',
        (variables: any) => {
            if (typeof variables !== 'object') {
                return false;
            }
            const keys = Object.keys(variables);
            for (let i = 0; i < keys.length; i++) {
                const key = keys[i];
                const value = variables[key];
                if (typeof value !== 'object') {
                    return false;
                }
                const valueKeys = Object.keys(value);
                for (let j = 0; j < valueKeys.length; j++) {
                    const valueKey = valueKeys[j];
                    if (valueKey === 'label' || valueKey === 'helperText' || valueKey === 'yup' || valueKey === 'defaultValue') {
                        continue;
                    }
                    return false;
                }
            }
            return true;
        }
    ),
    yup: inputYup.notRequired().default(undefined),
})

export const quantityBoxStandardInputForm = yup.object().shape({
    defaultValue: yup.number().notRequired().default(undefined),
    min: yup.number().notRequired().default(undefined),
    max: yup.number().notRequired().default(undefined),
    step: yup.number().notRequired().default(undefined),
    yup: inputYup.notRequired().default(undefined),
})

export const radioStandardInputForm = yup.object().shape({
    // Default can be of any type
    defaultValue: yup.mixed().notRequired().default(undefined),
    // Array of objects with keys of the format: { label: string, value: any }
    options: yup.array().of(
        yup.object().shape({
            label: yup.string().required(requiredErrorMessage),
            value: yup.mixed().required(requiredErrorMessage),
        })
    ).notRequired().default(undefined),
    // Display as row or column
    row: yup.boolean().notRequired().default(undefined),
    yup: inputYup.notRequired().default(undefined),
})

export const checkboxStandardInputForm = yup.object().shape({
    // Default is an array of boolean values
    defaultValue: yup.array().of(yup.boolean()).notRequired().default(undefined),
    // Array of { label: string }
    options: yup.array().of(
        yup.object().shape({
            label: yup.string().required(requiredErrorMessage),
        })
    ).notRequired().default(undefined),
    // Display as row or column
    row: yup.boolean().notRequired().default(undefined),
    yup: inputYup.notRequired().default(undefined),
})

export const switchStandardInputForm = yup.object().shape({
    defaultValue: yup.boolean().notRequired().default(undefined),
})

export const markdownStandardInputForm = yup.object().shape({
    defaultValue: yup.string().notRequired().default(undefined),
    minRows: yup.number().notRequired().default(undefined),
})
