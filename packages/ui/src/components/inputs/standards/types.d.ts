import { InputType } from "@local/shared";
import { CheckboxProps, DropzoneProps, FieldData, FieldDataBase, JsonProps, MarkdownProps, QuantityBoxProps, RadioProps, SwitchProps, TextFieldProps } from "forms/types";

/**
 * Handles all parts of creating a standard input, from props, to labels, to validation
 */
export interface BaseStandardInputProps {
    fieldName?: string;
    inputType: InputType;
    isEditing: boolean;
    label?: FieldDataBase['label'];
    onChange: (newSchema: FieldData) => void;
    schema: FieldData | null;
    storageKey: string;
    yup?: FieldData['yup'];
}

/**
 * Used for components that set the "prop" values of a standard input.
 */
interface StandardInputCommonProps<InputType> extends Omit<BaseStandardInputProps, 'fieldName' | 'inputType' | 'label' | 'onChange' | 'schema' | 'storageKey' | 'yup'> {
    onPropsChange: (newProps: Partial<InputType>) => void;
}

export interface CheckboxStandardInputProps extends StandardInputCommonProps<CheckboxProps> {
    color?: CheckboxProps['color'];
    defaultValue?: CheckboxProps['defaultValue'];
    options: CheckboxProps['options'];
    row?: CheckboxProps['row'];
};

export interface DropzoneStandardInputProps extends StandardInputCommonProps<DropzoneProps> {
    defaultValue?: DropzoneProps['defaultValue'];
};

export interface JsonStandardInputProps extends StandardInputCommonProps<JsonProps> {
    defaultValue?: JsonProps['defaultValue'];
    format?: JsonProps['format'];
    variables?: JsonProps['variables'];
};

export interface MarkdownStandardInputProps extends StandardInputCommonProps<MarkdownProps> {
    defaultValue?: MarkdownProps['defaultValue'];
};

export interface QuantityBoxStandardInputProps extends StandardInputCommonProps<QuantityBoxProps> {
    defaultValue?: QuantityBoxProps['defaultValue'];
    max?: QuantityBoxProps['max'];
    min?: QuantityBoxProps['min'];
    step?: QuantityBoxProps['step'];
};

export interface RadioStandardInputProps extends StandardInputCommonProps<RadioProps> {
    defaultValue?: RadioProps['defaultValue'];
    options: RadioProps['options'];
    row?: RadioProps['row'];
};

export interface SwitchStandardInputProps extends StandardInputCommonProps<SwitchProps> {
    defaultValue?: SwitchProps['defaultValue'];
    size?: SwitchProps['size'];
    color?: SwitchProps['color'];
};

export interface TextFieldStandardInputProps extends StandardInputCommonProps<TextFieldProps> {
    defaultValue?: TextFieldProps['defaultValue'];
    autoComplete?: TextFieldProps['autoComplete'];
    maxRows?: TextFieldProps['maxRows'];
};
