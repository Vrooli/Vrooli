/**
 * Validation schemas for all standards/inputs/outputs
 */
import { blankToUndefined, opt, optArr, req } from './base';
import * as yup from 'yup';

const required = yup.boolean();

export const inputYup = yup.object().shape({
    required: opt(required),
    type: yup.string().transform(blankToUndefined).oneOf(['string']).default('string'),
    // checks: 
})

export const textFieldStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.string().transform(blankToUndefined)),
    autoComplete: opt(yup.string().transform(blankToUndefined)),
    maxRows: opt(yup.number()),
    yup: opt(inputYup),
})

export const jsonStandardInputForm = yup.object().shape({
    format: req(yup.string().transform(blankToUndefined)),
    defaultValue: opt(yup.string().transform(blankToUndefined)),
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
    yup: opt(inputYup),
})

export const quantityBoxStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.number()),
    min: opt(yup.number()),
    max: opt(yup.number()),
    step: opt(yup.number()),
    yup: opt(inputYup),
})

export const radioStandardInputForm = yup.object().shape({
    // Default can be of any type
    defaultValue: opt(yup.mixed()),
    // Array of objects with keys of the format: { label: string, value: any }
    options: optArr(
        yup.object().shape({
            label: req(yup.string().transform(blankToUndefined)),
            value: req(yup.mixed()),
        })
    ),
    // Display as row or column
    row: opt(yup.boolean()),
    yup: opt(inputYup),
})

export const checkboxStandardInputForm = yup.object().shape({
    // Default is an array of boolean values
    defaultValue: optArr(yup.boolean()),
    // Array of { label: string }
    options: optArr(
        yup.object().shape({
            label: req(yup.string().transform(blankToUndefined)),
        })
    ),
    // Display as row or column
    row: opt(yup.boolean()),
    yup: opt(inputYup),
})

export const switchStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.boolean()),
})

export const markdownStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.string().transform(blankToUndefined)),
    minRows: opt(yup.number()),
})
