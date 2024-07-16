/**
 * Validation schemas for all standards/inputs/outputs
 */
import * as yup from "yup";
import { opt, req } from "../utils";

const required = yup.boolean();

const inputYup = yup.object().shape({
    required: opt(required),
    type: yup.string().trim().removeEmptyString().oneOf(["string"]).default("string"),
    // checks: 
});

//TODO remove
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
