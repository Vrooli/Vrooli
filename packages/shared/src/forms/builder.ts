import * as Yup from "yup";
import { RoutineType, RunRoutine } from "../api/types.js";
import { InputType } from "../consts/model.js";
import { uuid } from "../id/uuid.js";
import { RoutineVersionConfig, defaultConfigFormInputMap, defaultConfigFormOutputMap } from "../run/configs/routine.js";
import { isObject } from "../utils/objects.js";
import { CheckboxFormInputProps, CodeFormInputProps, DropzoneFormInputProps, FormElement, FormInputBase, FormInputType, FormSchema, IntegerFormInputProps, LanguageFormInputProps, LinkItemFormInputProps, LinkUrlFormInputProps, RadioFormInputProps, SelectorFormInputOption, SelectorFormInputProps, SliderFormInputProps, SwitchFormInputProps, TagSelectorFormInputProps, TextFormInputProps, YupField, YupType } from "./types.js";

const DEFAULT_SLIDER_MIN = 0;
const DEFAULT_SLIDER_MAX = 100;
const DEFAULT_SLIDER_STEP = 20;

function isNumeric(n: any) {
    return !isNaN(parseFloat(n)) && isFinite(n);
}

function nearest(value: number, min: number, max: number, steps: number) {
    let zerone = Math.round((value - min) * steps / (max - min)) / steps; // bring to 0-1 range    
    zerone = Math.min(Math.max(zerone, 0), 1); // keep in range in case value is off limits
    return zerone * (max - min) + min;
}

/**
 * Maps a form input type to a function that sets/repairs its type-specific props
 * @returns The properly-shaped props for the given input type
 */
export const healFormInputPropsMap: { [key in InputType]: (props: any) => any } = {
    [InputType.Checkbox]: function healCheckboxProps(props: Partial<CheckboxFormInputProps>): CheckboxFormInputProps {
        return {
            color: "secondary",
            defaultValue: new Array(props.options?.length ?? 0).fill(false),
            options: [{
                label: "Option 1",
                value: "option-1",
            }],
            maxSelection: 0,
            minSelection: 0,
            row: false,
            ...props,
        } as const;
    },
    [InputType.Dropzone]: function healDropzoneProps(props: Partial<DropzoneFormInputProps>): DropzoneFormInputProps {
        return {
            defaultValue: [],
            ...props,
        } as const;
    },
    [InputType.JSON]: function healJsonProps(props: Partial<CodeFormInputProps>): CodeFormInputProps {
        return {
            defaultValue: "",
            ...props,
        } as const;
    },
    [InputType.LanguageInput]: function healLanguageInputProps(props: Partial<LanguageFormInputProps>): LanguageFormInputProps {
        return {
            defaultValue: [],
            ...props,
        } as const;
    },
    [InputType.LinkItem]: function healLinkItemProps(props: Partial<LinkItemFormInputProps>): LinkItemFormInputProps {
        return {
            defaultValue: "",
            limitTo: [],
            ...props,
        } as const;
    },
    [InputType.LinkUrl]: function healLinkUrlProps(props: Partial<LinkUrlFormInputProps>): LinkUrlFormInputProps {
        return {
            acceptedHosts: [],
            defaultValue: "",
            ...props,
        } as const;
    },
    [InputType.IntegerInput]: function healIntegerInputProps(props: Partial<IntegerFormInputProps>): IntegerFormInputProps {
        const max = (isNumeric(props.max) ? props.max : Number.MAX_SAFE_INTEGER) as number;
        const min = (isNumeric(props.min) ? props.min : Number.MIN_SAFE_INTEGER) as number;
        const step = (isNumeric(props.step) ? props.step : 1) as number;
        const defaultValue = (isNumeric(props.defaultValue) ? props.defaultValue : 0) as number;
        return {
            defaultValue,
            max,
            min,
            step,
            ...props,
        } as const;
    },
    [InputType.Radio]: (props: Partial<RadioFormInputProps>): RadioFormInputProps => ({
        defaultValue: (Array.isArray(props.options) && props.options.length > 0) ? props.options[0]!.value : "",
        options: [],
        ...props,
    }),
    [InputType.Selector]: (props: Partial<SelectorFormInputProps<any>>): SelectorFormInputProps<any> => ({
        options: [],
        getOptionDescription: (option: SelectorFormInputOption) =>
            typeof option === "object"
                && Object.prototype.hasOwnProperty.call(option, "description")
                && typeof option.description === "string"
                ? option.description
                : null,
        getOptionLabel: (option: SelectorFormInputOption) =>
            typeof option === "object"
                && Object.prototype.hasOwnProperty.call(option, "label")
                && typeof option.label === "string"
                ? option.label
                : null,
        getOptionValue: (option: SelectorFormInputOption) =>
            typeof option === "object"
                && Object.prototype.hasOwnProperty.call(option, "value")
                ? option.value
                : null,
        ...props,
    }),
    [InputType.Slider]: (props: Partial<SliderFormInputProps>): SliderFormInputProps => {
        const max = (isNumeric(props.max) ? props.max : DEFAULT_SLIDER_MAX) as number;
        const min = (isNumeric(props.min) ? props.min : DEFAULT_SLIDER_MIN) as number;
        const step = (isNumeric(props.step) ? props.step : (max - min) / DEFAULT_SLIDER_STEP) as number; // Default to 20 steps
        const defaultValue = (isNumeric(props.defaultValue) ? props.defaultValue : nearest((min + max) / 2, min, max, step)) as number;
        return {
            ...props, // Props go first this time because we're fixing invalid values
            defaultValue,
            min,
            max,
            step,
        };
    },
    [InputType.Switch]: (props: Partial<SwitchFormInputProps>): SwitchFormInputProps => ({
        defaultValue: false,
        color: "secondary",
        label: "",
        size: "medium",
        ...props,
    }),
    [InputType.TagSelector]: (props: Partial<TagSelectorFormInputProps>): TagSelectorFormInputProps => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.Text]: (props: Partial<TextFormInputProps>): TextFormInputProps => ({
        autoComplete: "off",
        defaultValue: "",
        isMarkdown: true,
        maxChars: 1000,
        maxRows: 2,
        minRows: 4,
        ...props,
    }),
};

/**
 * Maps input types to their yup types. 
 * If an input type is not in this object, then 
 * it cannot be validated with yup.
 */
export const InputToYupType: { [key in InputType]?: YupType } = {
    // [InputType.Checkbox]: 'array', //TODO
    [InputType.JSON]: "string",
    [InputType.IntegerInput]: "number",
    [InputType.Radio]: "string",
    [InputType.Selector]: "string",
    [InputType.Slider]: "number",
    [InputType.Switch]: "boolean",
    [InputType.Text]: "string",
};

export class FormBuilder {
    private static readonly INPUT_PREFIX = "input";
    private static readonly OUTPUT_PREFIX = "output";

    /**
     * Creates a Formik `initialValues` object from a form schema
     * 
     * @param elements The form schema elements
     * @param prefix Prefix for the field names, in case you need to store multiple 
     * element sets in one formik (e.g. both inputs and outputs)
     * @returns An object with keys for each input field in the elements array 
     * (i.e. removes headers and other non-input elements) and their default values
     */
    static generateInitialValues(
        elements: readonly FormElement[] | null | undefined,
        prefix?: string,
    ): Record<string, never> {
        if (!Array.isArray(elements)) return {};
        const result: Record<string, never> = {};
        // Loop through each element in the schema
        for (const element of elements) {
            // Skip non-input elements
            if (!Object.prototype.hasOwnProperty.call(element, "fieldName")) continue;
            const formInput = element as FormInputType;
            const key = prefix ? `${prefix}-${formInput.fieldName}` : formInput.fieldName;
            // If it exists in the heal map, pass it through and use the resulting default value
            if (formInput.type in healFormInputPropsMap) {
                result[key] = healFormInputPropsMap[formInput.type](formInput.props ?? {}).defaultValue as never;
            }
            // If not, try using the defaultValue prop directly
            else if (formInput.props?.defaultValue !== undefined) {
                result[key] = formInput.props.defaultValue as never;
            }
            // Otherwise, set it to an empty string. It's worse to have an undefined value than a
            // possibly incorrect value, at least according to Formike error messages
            else {
                result[key] = "" as never;
            }
        }
        return result;
    }

    /**
     * Generates a Formik `initialValues` object from a `RoutineVersionConfig`, 
     * including both input and output fields.
     * 
     * @param config The `RoutineVersionConfig` to generate initial values from
     * @param routineType The type of routine to generate initial values for
     * @param run Run data if we're in a run, or null if we're not
     * @returns An object with keys for each input and output field in the config
     */
    static generateInitialValuesFromRoutineConfig(
        config: RoutineVersionConfig,
        routineType: RoutineType,
        run?: Pick<RunRoutine, "io">,
    ): Record<string, never> {
        const formInput = config.formInput?.schema ?? defaultConfigFormInputMap[routineType]().schema;
        const formOutput = config.formOutput?.schema ?? defaultConfigFormOutputMap[routineType]().schema;

        const inputInitialValues = this.generateInitialValues(formInput.elements, this.INPUT_PREFIX);
        const outputInitialValues = this.generateInitialValues(formOutput.elements, this.OUTPUT_PREFIX);

        const initialValues = {
            ...inputInitialValues,
            ...outputInitialValues,
        };

        if (run && Array.isArray(run.io)) {
            for (const io of run.io) {
                if (io.routineVersionInput) {
                    const key = `${this.INPUT_PREFIX}-${io.routineVersionInput.name ?? io.routineVersionInput.id}`;
                    if (initialValues[key] !== undefined) {
                        try {
                            initialValues[key] = JSON.parse(io.data) as never;
                        } catch (error) {
                            console.error("Error parsing io.data", error);
                        }
                    }
                }
                if (io.routineVersionOutput) {
                    const key = `${this.OUTPUT_PREFIX}-${io.routineVersionOutput.name ?? io.routineVersionOutput.id}`;
                    if (initialValues[key] !== undefined) {
                        try {
                            initialValues[key] = JSON.parse(io.data) as never;
                        } catch (error) {
                            console.error("Error parsing io.data", error);
                        }
                    }
                }
            }
        }

        return initialValues;
    }

    /**
     * Generate a yup schema from a form schema. Each field in this schema 
     * contains its own yup schema, which we must combine into a single schema. 
     * Then we convert this schema into a yup object.
     * @param formSchema The schema of the entire form
     */
    static generateYupSchema(
        formSchema: Pick<FormSchema, "elements">,
        prefix?: string,
    ): Yup.ObjectSchema {
        if (!formSchema) return null;

        // Initialize an empty object to hold the field schemas
        const shape: { [fieldName: string]: Yup.AnySchema } = {};

        // Loop through each field in the form schema
        formSchema.elements.forEach(field => {
            // Skip non-input fields
            if (!("fieldName" in field)) return;
            const formInput = field as FormInputBase;
            const name = prefix ? `${prefix}-${formInput.fieldName}` : formInput.fieldName;

            // Field will only be validated if it has a yup schema, and it is a valid input type
            if (formInput.yup && InputToYupType[field.type]) {
                // Start building the Yup schema for this field
                let validator: Yup.AnySchema;

                // Determine the base Yup type based on InputToYupType
                const baseType = InputToYupType[field.type];

                if (!baseType) return; // Skip if type is not supported

                switch (baseType) {
                    case "string":
                        validator = Yup.string();
                        break;
                    case "number":
                        validator = Yup.number();
                        break;
                    case "boolean":
                        validator = Yup.boolean();
                        break;
                    case "date":
                        validator = Yup.date();
                        break;
                    case "object":
                        validator = Yup.object();
                        break;
                    default:
                        validator = Yup.mixed();
                        break;
                }

                // Apply required
                if (formInput.yup.required) {
                    validator = validator.required(`${field.label} is required`);
                } else {
                    validator = validator.nullable();
                }

                // Apply additional checks
                formInput.yup.checks.forEach(check => {
                    const { key, value, error } = check;
                    const method = (validator as any)[key];
                    if (typeof method === "function") {
                        if (value !== undefined && error !== undefined) {
                            validator = method.call(validator, value, error);
                        } else if (value !== undefined) {
                            validator = method.call(validator, value);
                        } else if (error !== undefined) {
                            validator = method.call(validator, error);
                        } else {
                            validator = method.call(validator);
                        }
                    } else {
                        throw new Error(`Validation method ${key} does not exist on Yup.${baseType}()`);
                    }
                });

                // Add the validator to the shape
                shape[name] = validator;
            }
        });

        // Build the Yup object schema
        const yupSchema = Yup.object().shape(shape);

        return yupSchema;
    }

    /**
     * Generate a Yup schema from a RoutineVersionConfig. It builds the schema for both input and output,
     * then concatenates them into a single schema.
     *
     * @param config The RoutineVersionConfig to generate the Yup schema from.
     * @param routineType The type of routine to generate the Yup schema for.
     * @returns A Yup object schema representing the combined validation for input and output fields.
     */
    static generateYupSchemaFromRoutineConfig(
        config: RoutineVersionConfig,
        routineType: RoutineType,
    ) {
        // Use the routine's formInput and formOutput schema, falling back to defaults if needed
        const formInput = config.formInput?.schema ?? defaultConfigFormInputMap[routineType]().schema;
        const formOutput = config.formOutput?.schema ?? defaultConfigFormOutputMap[routineType]().schema;

        // Generate the Yup schemas for inputs and outputs with the respective prefixes
        const inputSchema = (this.generateYupSchema(formInput, this.INPUT_PREFIX) || Yup.object());
        const outputSchema = (this.generateYupSchema(formOutput, this.OUTPUT_PREFIX) || Yup.object());

        // Combine the field definitions manually
        const combinedFields = {
            ...inputSchema.fields,
            ...outputSchema.fields,
        };

        return Yup.object().shape(combinedFields);
    }
}

export type CreateFormInputProps = Omit<Partial<FormInputType>, "props" | "type" | "yup"> & {
    props: Partial<FormInputType["props"]> | string | null | undefined;
    type: FormInputType["type"]; // Required
    yup: Partial<FormInputType["yup"]> | string | null | undefined;
}

/**
 * Creates FormInputType for a given input type, which may have stringified values 
 * if it's coming from the server
 * @param type The input type
 * @returns A FormInputType object with default values
 */
export function createFormInput({
    fieldName,
    id,
    label,
    props,
    type,
    yup,
    ...rest
}: CreateFormInputProps): FormInputType | null {
    // Return null if the type is invalid
    if (typeof type !== "string" || !healFormInputPropsMap[type]) return null;
    // Non-primitive props might be stringified from the server, so we need to parse them
    try {
        if (typeof props === "string") {
            const parsedProps = JSON.parse(props ?? "{}");
            props = isObject(parsedProps) ? parsedProps : {};
        }
        if (typeof yup === "string") {
            const parsedYup = JSON.parse(yup ?? "{}");
            yup = isObject(parsedYup) ? parsedYup : {};
        }
    } catch (error) {
        console.error("Error parsing props/yup", error);
        return null;
    }
    // Handle fallbacks
    if (!props) {
        props = {};
    }
    if (!yup) {
        yup = ({ checks: [] });
    }
    // Return the FormInputType object
    return ({
        type,
        props: healFormInputPropsMap[type](props),
        fieldName: fieldName ?? "",
        id: id ?? uuid(),
        label: label ?? "",
        yup: yup as YupField,
        ...rest,
    });
}

const FIELD_NAME_DELIMITER = "-";

/**
 * Creates the formik field name based on a fieldName and prefix
 * @param fieldName The field name to use
 * @param prefix The prefix to use
 * @returns The formik field name
 */
export function getFormikFieldName(fieldName: string, prefix?: string): string {
    return prefix ? `${prefix}${FIELD_NAME_DELIMITER}${fieldName}` : fieldName;
}
