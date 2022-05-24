import { FieldDataCheckbox, FieldDataDropzone, FieldDataJSON, FieldDataMarkdown, FieldDataQuantityBox, FieldDataRadio, FieldDataSwitch, FieldDataTextField } from "forms/types";

export interface BaseStandardInputProps<FieldDataType> {
    isEditing: boolean;
    onChange: (newSchema: FieldDataType) => void;
    schema: FieldDataType | null;
}

export interface CheckboxStandardInputProps extends BaseStandardInputProps<FieldDataCheckbox> {};
export interface DropzoneStandardInputProps extends BaseStandardInputProps<FieldDataDropzone> {};
export interface JsonStandardInputProps extends BaseStandardInputProps<FieldDataJSON> {};
export interface MarkdownStandardInputProps extends BaseStandardInputProps<FieldDataMarkdown> {};
export interface QuantityBoxStandardInputProps extends BaseStandardInputProps<FieldDataQuantityBox> {};
export interface RadioStandardInputProps extends BaseStandardInputProps<FieldDataRadio> {};
export interface SwitchStandardInputProps extends BaseStandardInputProps<FieldDataSwitch> {};
export interface TextFieldStandardInputProps extends BaseStandardInputProps<FieldDataTextField> {};