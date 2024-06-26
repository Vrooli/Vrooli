import { InputType, isObject, uuid } from "@local/shared";
import { CheckboxFormInputProps, CodeFormInputProps, DropzoneFormInputProps, FormInputType, IntegerFormInputProps, LanguageFormInputProps, LinkItemFormInputProps, LinkUrlFormInputProps, RadioFormInputProps, SelectorFormInputOption, SelectorFormInputProps, SliderFormInputProps, SwitchFormInputProps, TagSelectorFormInputProps, TextFormInputProps, YupField } from "forms/types";

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
    [InputType.Checkbox]: (props: Partial<CheckboxFormInputProps>): CheckboxFormInputProps => ({
        color: "secondary",
        defaultValue: new Array(props.options?.length ?? 0).fill(false),
        options: [],
        maxSelection: 0,
        minSelection: 0,
        row: false,
        ...props,
    }),
    [InputType.Dropzone]: (props: Partial<DropzoneFormInputProps>): DropzoneFormInputProps => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.JSON]: (props: Partial<Omit<CodeFormInputProps, "id">>): Omit<CodeFormInputProps, "id" | "name"> => ({
        defaultValue: "",
        ...props,
    }),
    [InputType.LanguageInput]: (props: Partial<LanguageFormInputProps>): LanguageFormInputProps => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.LinkItem]: (props: Partial<LinkItemFormInputProps>): LinkItemFormInputProps => ({
        defaultValue: "",
        limitTo: [],
        ...props,
    }),
    [InputType.LinkUrl]: (props: Partial<LinkUrlFormInputProps>): LinkUrlFormInputProps => ({
        acceptedHosts: [],
        defaultValue: "",
        ...props,
    }),
    [InputType.IntegerInput]: (props: Partial<Omit<IntegerFormInputProps, "name">>): Omit<IntegerFormInputProps, "name"> => ({
        defaultValue: 0,
        max: Number.MAX_SAFE_INTEGER,
        min: Number.MIN_SAFE_INTEGER,
        step: 1,
        ...props,
    }),
    [InputType.Radio]: (props: Partial<RadioFormInputProps>): RadioFormInputProps => ({
        defaultValue: (Array.isArray(props.options) && props.options.length > 0) ? props.options[0].value : "",
        options: [],
        ...props,
    }),
    [InputType.Selector]: (props: Partial<SelectorFormInputProps<any>>): Omit<SelectorFormInputProps<any>, "name"> => ({
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
 * Populates a form input array with unset default values
 * @param fields The form's field data
 */
export function generateDefaultProps(fields: FormInputType[]): FormInputType[] {
    if (!Array.isArray(fields)) return [];
    // Remove invalid types
    let result = fields.filter(field => field.type in healFormInputPropsMap);
    // Heal each field
    result = result.map(field => {
        const { props, ...otherKeys } = field;
        return {
            props: healFormInputPropsMap[field.type](props as any),
            ...otherKeys,
        };
    });
    // Return the result
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
