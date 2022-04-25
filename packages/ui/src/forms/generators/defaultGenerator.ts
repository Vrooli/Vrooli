import { CheckboxProps, InputType, RadioProps, SliderProps, SwitchProps, TextFieldProps } from '../types';
import { FieldData } from 'forms/types';
import { DropzoneProps, QuantityBoxProps, SelectorProps } from 'components/inputs/types';

/**
 * Maps a data input type to a function that calculates its default values.
 * Values already set have precedence
 * @returns The passed-in props object with default values added
 */
const defaultMap: { [key in InputType]: (props: any) => any } = {
    [InputType.Checkbox]: (props: CheckboxProps): CheckboxProps => ({
        defaultValue: false,
        color: 'secondary',
        ...props
    }),
    [InputType.Dropzone]: (props: DropzoneProps): DropzoneProps => ({ ...props }),
    [InputType.Radio]: (props: RadioProps) => ({
        defaultValue: (Array.isArray(props.options) && props.options.length > 0) ? props.options[0].value : '',
        ...props
    }),
    [InputType.Selector]: (props: SelectorProps): SelectorProps => ({ ...props }),
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
    [InputType.Switch]: (props: SwitchProps): SwitchProps => ({
        defaultValue: false,
        color: 'secondary',
        size: 'medium',
        ...props
    }),
    [InputType.TextField]: (props: TextFieldProps): TextFieldProps => ({
        defaultValue: '',
        ...props
    }),
    [InputType.QuantityBox]: (props: Omit<QuantityBoxProps, 'id' | 'value' | 'handleChange'>): Omit<QuantityBoxProps, 'id' | 'value' | 'handleChange'> => ({
        ...props
    }),
}

/**
 * Populates a form's field data with unset default values
 * @param fields The form's field data
 */
export const generateDefaultProps = (fields: FieldData[]): FieldData[] => {
    console.log('generating default props', fields)
    if (!fields) return [];
    return fields.map(field => {
        const { props, ...otherKeys } = field;
        return {
            props: defaultMap[field.type](props as any),
            ...otherKeys
        }
    });
}