import { FormHeaderType, FormInputType } from "@local/shared";

export interface FormDividerProps {
    isEditing: boolean;
    onDelete: () => unknown;
}

export interface FormHeaderProps {
    element: FormHeaderType;
    isEditing: boolean;
    onUpdate: (data: Partial<FormHeaderType>) => unknown;
    onDelete: () => unknown;
}

export interface FormInputProps<FieldData extends FormInputType = FormInputType> {
    disabled?: boolean;
    fieldData: FieldData;
    /** 
     * Prefix for the field names, in case you need to store multiple 
     * element sets in one formik (e.g. both inputs and outputs) 
     */
    fieldNamePrefix?: string;
    index: number;
    isEditing: boolean;
    /** Provide when building form to configure input properties */
    onConfigUpdate: (fieldData: FieldData) => unknown;
    /** Provide when building form to delete input element */
    onDelete: () => unknown;
    textPrimary?: string;
}
