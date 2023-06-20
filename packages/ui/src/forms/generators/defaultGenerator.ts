import { InputType } from "@local/shared";
import { StandardLanguage } from "components/inputs/CodeInputBase/CodeInputBase";
import { CodeProps, FieldData } from "forms/types";
import { CheckboxProps, DropzoneProps, IntegerInputProps, LanguageInputProps, MarkdownProps, RadioProps, SelectorProps, SliderProps, SwitchProps, TagSelectorProps, TextFieldProps, YupField } from "../types";

/**
 * Maps a data input type to a function that calculates its default values.
 * Values already set have precedence. 
 * Assumes that any key in props might be missing.
 * @returns The passed-in props object with default values added
 */
export const defaultStandardPropsMap: { [key in InputType]: (props: any) => any } = {
    [InputType.Checkbox]: (props: Partial<CheckboxProps>): CheckboxProps => ({
        color: "secondary",
        defaultValue: new Array(props.options?.length ?? 0).fill(false),
        options: [],
        row: true,
        ...props,
    }),
    [InputType.Dropzone]: (props: Partial<DropzoneProps>): DropzoneProps => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.JSON]: (props: Partial<Omit<CodeProps, "id">>): Omit<CodeProps, "id"> => ({
        defaultValue: "",
        limitTo: [StandardLanguage.JsonStandard],
        ...props,
    }),
    [InputType.LanguageInput]: (props: Partial<LanguageInputProps>): LanguageInputProps => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.IntegerInput]: (props: Partial<Omit<IntegerInputProps, "name">>): Omit<IntegerInputProps, "name"> => ({
        ...props,
    }),
    [InputType.Markdown]: (props: Partial<Omit<MarkdownProps, "id">>): Omit<MarkdownProps, "id" | "name"> => ({
        defaultValue: "",
        ...props,
    }),
    [InputType.Prompt]: (props: Partial<Omit<MarkdownProps, "id">>): Omit<MarkdownProps, "id" | "name"> => ({
        defaultValue: "",
        ...props,
    }), //TODO change when custom prompt input implemented, if it's needed
    [InputType.Radio]: (props: Partial<RadioProps>) => ({
        defaultValue: (Array.isArray(props.options) && props.options.length > 0) ? props.options[0].value : "",
        ...props,
    }),
    [InputType.Selector]: (props: Partial<SelectorProps<any>>): Omit<SelectorProps<any>, "name"> => ({
        options: [],
        getOptionLabel: (option: any) => option,
        ...props,
    }),
    [InputType.Slider]: (props: SliderProps) => {
        let { defaultValue, min, max, step, ...otherProps } = props;
        const isNumeric = (n: any) => !isNaN(parseFloat(n)) && isFinite(n);
        const nearest = (value: number, min: number, max: number, steps: number) => {
            let zerone = Math.round((value - min) * steps / (max - min)) / steps; // bring to 0-1 range    
            zerone = Math.min(Math.max(zerone, 0), 1); // keep in range in case value is off limits
            return zerone * (max - min) + min;
        };
        if (!isNumeric(min)) min = 0;
        if (!isNumeric(max)) max = 100;
        if (!isNumeric(step)) step = (max - min) / 20; // Default to 20 steps
        if (!isNumeric(defaultValue)) defaultValue = nearest((min + max) / 2, min, max, step || 1);
        return {
            defaultValue,
            min,
            max,
            step,
            ...otherProps,
        };
    },
    [InputType.Switch]: (props: Partial<SwitchProps>): SwitchProps => ({
        defaultValue: false,
        color: "secondary",
        size: "medium",
        ...props,
    }),
    [InputType.TagSelector]: (props: Partial<TagSelectorProps>): TagSelectorProps => ({
        defaultValue: [],
        ...props,
    }),
    [InputType.TextField]: (props: Partial<TextFieldProps>): TextFieldProps => ({
        autoComplete: "off",
        defaultValue: "",
        maxRows: 1,
        minRows: 1,
        ...props,
    }),
};

/**
 * Populates a FieldData array with unset default values
 * @param fields The form's field data
 */
export const generateDefaultProps = (fields: FieldData[]): FieldData[] => {
    if (!fields) return [];
    return fields.map(field => {
        const { props, ...otherKeys } = field;
        return {
            props: defaultStandardPropsMap[field.type](props as any),
            ...otherKeys,
        };
    });
};

interface CreateDefaultFieldDataProps {
    fieldName?: string;
    label?: string;
    type: InputType;
    yup?: YupField
}
/**
 * Creates default FieldData for a given input type
 * @param type The input type
 * @returns A FieldData object with default values
 */
export const createDefaultFieldData = ({
    fieldName,
    label,
    type,
    yup,
}: CreateDefaultFieldDataProps): FieldData | null => {
    if (!type || !defaultStandardPropsMap[type]) return null;
    return ({
        type: type as any,
        props: defaultStandardPropsMap[type]({}),
        fieldName: fieldName ?? "",
        label: label ?? "",
        yup: yup ?? ({
            checks: [],
        }),
    });
};
