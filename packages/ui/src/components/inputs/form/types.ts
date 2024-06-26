import { Theme } from "@mui/material";
import { FormHeaderType, FormInputType, GridContainer, GridContainerBase } from "forms/types";

export interface GeneratedGridProps {
    childContainers?: GridContainer[];
    fields: FormInputType[];
    layout?: GridContainer | GridContainerBase;
    theme: Theme;
}

export interface FormHeaderProps {
    element: FormHeaderType;
    isSelected: boolean;
    onUpdate: (data: Partial<FormHeaderType>) => unknown;
    onDelete: () => void;
}

export interface FormInputProps<FieldData extends FormInputType = FormInputType> {
    copyInput?: (fieldName: string) => unknown;
    disabled?: boolean;
    fieldData: FieldData;
    index: number;
    /** Provide when building form to configure input properties */
    onConfigUpdate?: (fieldData: FieldData) => unknown;
    /** Provide when building form to delete input element */
    onDelete?: () => unknown;
    textPrimary?: string;
}
