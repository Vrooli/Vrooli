import type { BoxProps } from "@mui/material";
import type { FormHeaderType, FormStructureType, IconInfo, InputType } from "@vrooli/shared";

export interface ElementBuildOuterBoxProps extends BoxProps {
    isSelected: boolean;
}

export interface ToolbarBoxProps extends BoxProps {
    formElementsCount: number;
}

export type PopoverListInput = {
    category: string;
    items: readonly {
        type: InputType;
        iconInfo: IconInfo | null;
        label: string
    }[];
}[]

export type PopoverListStructure = {
    category: string;
    items: readonly {
        type: FormStructureType;
        iconInfo: IconInfo | null;
        label: string;
        tag?: FormHeaderType["tag"];
    }[];
}[]

export interface PopoverListItemProps {
    iconInfo: IconInfo | null;
    label: string;
    tag?: FormHeaderType["tag"];
    type: FormStructureType | InputType;
    onAddHeader: (headerData: Partial<FormHeaderType>) => unknown;
    onAddInput: (inputData: Omit<Partial<FormInputType>, "type"> & { type: InputType; }) => unknown;
    onAddStructure: (structureData: Omit<Partial<FormInformationalType>, "type"> & { type: "Divider" | "Image" | "QrCode" | "Tip" | "Video" }) => unknown;
}

export type GeneratedFormItemProps = {
    /** The child form input or form structure element */
    children: JSX.Element | null | undefined;
    /** The number of fields in the grid, for calculating grid item size */
    fieldsInGrid: number;
    /** Whether to wrap the child in a grid item */
    isInGrid: boolean;
}

// Re-export shared types that are used in multiple files
export type { FormInformationalType, FormInputType } from "@vrooli/shared";
