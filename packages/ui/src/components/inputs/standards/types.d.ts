import { InputType } from "@local/shared";
import { FieldDataCheckbox, FieldDataDropzone, FieldDataJSON, FieldDataMarkdown, FieldDataQuantityBox, FieldDataRadio, FieldDataSwitch, FieldDataTextField } from "forms/types";

export interface BaseStandardInputProps<FieldDataType> {
    fieldName?: string;
    inputType: InputType;
    isEditing: boolean;
    onChange: (newSchema: FieldDataType) => void;
    schema: FieldDataType | null;
    storageKey: string;
}

// While base component can take null schema, actual components cannot
interface StandardInputBase<FieldDataType> extends BaseStandardInputProps<FieldDataType> { schema: FieldDataType; }

export interface CheckboxStandardInputProps extends StandardInputBase<FieldDataCheckbox> {};
export interface DropzoneStandardInputProps extends StandardInputBase<FieldDataDropzone> {};
export interface JsonStandardInputProps extends StandardInputBase<FieldDataJSON> {};
export interface MarkdownStandardInputProps extends StandardInputBase<FieldDataMarkdown> {};
export interface QuantityBoxStandardInputProps extends StandardInputBase<FieldDataQuantityBox> {};
export interface RadioStandardInputProps extends StandardInputBase<FieldDataRadio> {};
export interface SwitchStandardInputProps extends StandardInputBase<FieldDataSwitch> {};
export interface TextFieldStandardInputProps extends StandardInputBase<FieldDataTextField> {};