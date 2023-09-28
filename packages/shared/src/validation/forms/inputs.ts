/**
 * Validation schemas for all standards/inputs/outputs
 */
import * as yup from "yup";
import { opt, optArr, req } from "../utils";

const required = yup.boolean();

export const inputYup = yup.object().shape({
    required: opt(required),
    type: yup.string().trim().removeEmptyString().oneOf(["string"]).default("string"),
    // checks: 
});

export const textStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.string().trim().removeEmptyString()),
    autoComplete: opt(yup.string().trim().removeEmptyString()),
    isMarkdown: opt(yup.boolean()),
    maxRows: opt(yup.number()),
    yup: opt(inputYup),
});

export const jsonStandardInputForm = yup.object().shape({
    format: req(yup.string().trim().removeEmptyString()),
    defaultValue: opt(yup.string().trim().removeEmptyString()),
    // Object with keys of the format: { label?: string, helperText?: string, yup?: inputYup, defaultValue?: string | object }
    variables: yup.object().test(
        "variables",
        "Variables must be an object with keys of the format { label?: string, helperText?: string, yup?: inputYup, defaultValue?: string | object }",
        (variables: any) => {
            if (typeof variables !== "object") {
                return false;
            }
            const keys = Object.keys(variables);
            for (let i = 0; i < keys.length; i++) {
                const key = keys[i];
                if (typeof key !== "string") {
                    return false;
                }
                const value = variables[key];
                if (typeof value !== "object") {
                    return false;
                }
                const valueKeys = Object.keys(value);
                for (let j = 0; j < valueKeys.length; j++) {
                    const valueKey = valueKeys[j];
                    if (valueKey === "label" || valueKey === "helperText" || valueKey === "yup" || valueKey === "defaultValue") {
                        continue;
                    }
                    return false;
                }
            }
            return true;
        },
    ),
    yup: opt(inputYup),
});

export const quantityBoxStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.number()),
    min: opt(yup.number()),
    max: opt(yup.number()),
    step: opt(yup.number()),
    yup: opt(inputYup),
});

export const radioStandardInputForm = yup.object().shape({
    // Default can be of any type
    defaultValue: opt(yup.mixed()),
    // Array of objects with keys of the format: { label: string, value: any }
    options: optArr(
        yup.object().shape({
            label: req(yup.string().trim().removeEmptyString()),
            value: req(yup.mixed()),
        }),
    ),
    // Display as row or column
    row: opt(yup.boolean()),
    yup: opt(inputYup),
});

export const checkboxStandardInputForm = yup.object().shape({
    // Default is an array of boolean values
    defaultValue: optArr(yup.boolean()),
    // Array of { label: string }
    options: optArr(
        yup.object().shape({
            label: req(yup.string().trim().removeEmptyString()),
        }),
    ),
    // Display as row or column
    row: opt(yup.boolean()),
    yup: opt(inputYup),
});

export const switchStandardInputForm = yup.object().shape({
    defaultValue: opt(yup.boolean()),
});
