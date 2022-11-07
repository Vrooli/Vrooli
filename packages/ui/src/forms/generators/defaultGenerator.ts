import { CheckboxProps, DropzoneProps, JSONProps, MarkdownProps, RadioProps, SelectorProps, SliderProps, SwitchProps, TextFieldProps, QuantityBoxProps, TagSelectorProps, LanguageInputProps, YupField } from '../types';
import { FieldData } from 'forms/types';
import { InputType } from '@shared/consts';

/**
 * Maps a data input type to a function that calculates its default values.
 * Values already set have precedence. 
 * Assumes that any key in props might be missing.
 * @returns The passed-in props object with default values added
 */
const defaultMap: { [key in InputType]: (props: any) => any } = {
    [InputType.Checkbox]: (props: Partial<CheckboxProps>): CheckboxProps => ({
        color: 'secondary',
        defaultValue: new Array(props.options?.length?? 0).fill(false),
        options: [],
        row: true,
        ...props
    }),
    [InputType.Dropzone]: (props: Partial<DropzoneProps>): DropzoneProps => ({
        defaultValue: [],
        ...props
    }),
    [InputType.JSON]: (props: Partial<Omit<JSONProps, 'id'>>): Omit<JSONProps, 'id'> => ({
        defaultValue: '',
        ...props
    }),
    [InputType.LanguageInput]: (props: Partial<LanguageInputProps>): LanguageInputProps => ({
        defaultValue: [],
        ...props
    }),
    [InputType.Markdown]: (props: Partial<Omit<MarkdownProps, 'id'>>): Omit<MarkdownProps, 'id'> => ({
        defaultValue: '',
        ...props
    }),
    [InputType.Radio]: (props: Partial<RadioProps>) => ({
        defaultValue: (Array.isArray(props.options) && props.options.length > 0) ? props.options[0].value : '',
        ...props
    }),
    [InputType.Selector]: (props: Partial<SelectorProps<any>>): SelectorProps<any> => ({ 
        options: [],
        getOptionLabel: (option: any) => option,
        ...props 
    }),
    [InputType.Slider]: (props: SliderProps) => {
        let { defaultValue, min, max, step, ...otherProps } = props;
        const isNumeric = (n: any) => !isNaN(parseFloat(n)) && isFinite(n);
        const nearest = (value: number, min: number, max: number, steps: number) => {
            var zerone = Math.round((value - min) * steps / (max - min)) / steps; // bring to 0-1 range    
            zerone = Math.min(Math.max(zerone, 0), 1) // keep in range in case value is off limits
            return zerone * (max - min) + min;
        }
        if (!isNumeric(min)) min = 0;
        if (!isNumeric(max)) max = 100;
        if (!isNumeric(step)) step = (max - min) / 20; // Default to 20 steps
        if (!isNumeric(defaultValue)) defaultValue = nearest((min + max) / 2, min, max, step || 1);
        return {
            defaultValue,
            min,
            max,
            step,
            ...otherProps
        }
    },
    [InputType.Switch]: (props: Partial<SwitchProps>): SwitchProps => ({
        defaultValue: false,
        color: 'secondary',
        size: 'medium',
        ...props
    }),
    [InputType.TagSelector]: (props: Partial<TagSelectorProps>): TagSelectorProps => ({
        defaultValue: [],
        ...props
    }),
    [InputType.TextField]: (props: Partial<TextFieldProps>): TextFieldProps => ({
        defaultValue: '',
        ...props
    }),
    [InputType.QuantityBox]: (props: Partial<Omit<QuantityBoxProps, 'id' | 'value' | 'handleChange'>>): Omit<QuantityBoxProps, 'id' | 'value' | 'handleChange'> => ({
        ...props
    }),
}

/**
 * Populates a FieldData array with unset default values
 * @param fields The form's field data
 */
export const generateDefaultProps = (fields: FieldData[]): FieldData[] => {
    if (!fields) return [];
    return fields.map(field => {
        const { props, ...otherKeys } = field;
        return {
            props: defaultMap[field.type](props as any),
            ...otherKeys
        }
    });
}

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
    yup
}: CreateDefaultFieldDataProps): FieldData | null => {
    if (!type || !defaultMap[type]) return null;
    return ({
        type,
        props: defaultMap[type]({}),
        fieldName: fieldName ?? '',
        label: label ?? '',
        yup: yup ?? ({
            checks: [],
        }),
    })
}