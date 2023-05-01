import { Theme } from "@mui/material";
import { FieldData, GridContainer, GridContainerBase } from "forms/types";

export interface GeneratedGridProps {
    childContainers?: GridContainer[];
    fields: FieldData[];
    layout?: GridContainer | GridContainerBase;
    onUpload: (fieldName: string, files: string[]) => void;
    theme: Theme;
    zIndex: number;
}

export interface GeneratedInputComponentProps {
    disabled?: boolean;
    fieldData: FieldData;
    index?: number;
    onUpload: (fieldName: string, files: string[]) => void;
    zIndex: number;
}

export interface GeneratedInputComponentWithLabelProps extends GeneratedInputComponentProps {
    copyInput?: (fieldName: string) => void;
    textPrimary: string;
}
