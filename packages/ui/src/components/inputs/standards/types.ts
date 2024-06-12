import { InputType } from "@local/shared";
import { FieldData, FieldDataBase } from "forms/types";
import { CodeLanguage } from "../CodeInput/CodeInput";

/**
 * Handles all parts of creating a standard input, from props, to labels, to validation
 */
export interface BaseStandardInputProps {
    fieldName: string;
    inputType: InputType;
    isEditing: boolean;
    label?: FieldDataBase["label"];
    storageKey: string;
    yup?: FieldData["yup"];
}

/**
 * Used for components that set the "prop" values of a standard input.
 */
type StandardInputCommonProps = Omit<BaseStandardInputProps, "inputType" | "label" | "storageKey" | "yup">

export type CheckboxStandardInputProps = StandardInputCommonProps;

export type CodeStandardInputProps = StandardInputCommonProps & {
    limitTo?: CodeLanguage[];
};

export type DropzoneStandardInputProps = StandardInputCommonProps;

export type IntegerStandardInputProps = StandardInputCommonProps;

export type RadioStandardInputProps = StandardInputCommonProps;

export type SwitchStandardInputProps = StandardInputCommonProps;

export type TextStandardInputProps = StandardInputCommonProps;

export interface StandardInputProps {
    disabled?: boolean;
    fieldName: string;
}
