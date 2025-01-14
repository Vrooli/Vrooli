import { InputType, isObject, uuid } from "@local/shared";
import { CheckboxFormInputProps, CodeFormInputProps, DropzoneFormInputProps, FormElement, FormInputType, IntegerFormInputProps, LanguageFormInputProps, LinkItemFormInputProps, LinkUrlFormInputProps, RadioFormInputProps, SelectorFormInputOption, SelectorFormInputProps, SliderFormInputProps, SwitchFormInputProps, TagSelectorFormInputProps, TextFormInputProps, YupField } from "./types";

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
 * Creates a Formik `initialValues` object from a form schema
 * @param elements The form schema elements
 * @param prefix Prefix for the field names, in case you need to store multiple 
 * element sets in one formik (e.g. both inputs and outputs)
 * @returns An object with keys for each input field in the elements array 
 * (i.e. removes headers and other non-input elements) and their default values
 */
export function generateInitialValues(
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
