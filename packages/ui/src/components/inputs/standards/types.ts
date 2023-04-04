import { InputType } from "@shared/consts";
import { FieldData, FieldDataBase } from "forms/types";

/**
 * Handles all parts of creating a standard input, from props, to labels, to validation
 */
export interface BaseStandardInputProps {
    fieldName?: string;
    inputType: InputType;
    isEditing: boolean;
    label?: FieldDataBase['label'];
    storageKey: string;
    yup?: FieldData['yup'];
}

/**
 * Used for components that set the "prop" values of a standard input.
 */
interface StandardInputCommonProps extends Omit<BaseStandardInputProps, 'inputType' | 'label' | 'storageKey' | 'yup'> { }

export interface CheckboxStandardInputProps extends StandardInputCommonProps { };

export interface DropzoneStandardInputProps extends StandardInputCommonProps { };

export interface JsonStandardInputProps extends StandardInputCommonProps { };

export interface MarkdownStandardInputProps extends StandardInputCommonProps { };

export interface IntegerStandardInputProps extends StandardInputCommonProps { };

export interface RadioStandardInputProps extends StandardInputCommonProps { };

export interface SwitchStandardInputProps extends StandardInputCommonProps { };

export interface TextFieldStandardInputProps extends StandardInputCommonProps { };

export interface StandardInputProps {
    disabled?: boolean;
    fieldName: string;
    zIndex: number;
}