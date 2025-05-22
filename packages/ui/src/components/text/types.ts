import { type ListObject, type Routine } from "@local/shared";
import { type BoxProps } from "@mui/material";
import { type IconInfo } from "../../icons/Icons.js";
import { type SxType } from "../../types.js";
import { type DisplayAdornment } from "../../utils/display/listTools.js";
import { type ObjectType } from "../../utils/navigation/openObject.js";

export interface DateDisplayProps extends Omit<BoxProps, "zIndex"> {
    loading?: boolean;
    showDateAndTime?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
}

export interface OwnerLabelProps {
    confirmOpen?: (callback: () => unknown) => unknown;
    language?: string
    objectType: ObjectType;
    owner: Routine["owner"] | null | undefined
    sxs?: {
        label?: SxType;
    }
}

export interface StatsCompactProps<T extends ListObject> {
    handleObjectUpdate: (object: T) => unknown;
    object: T | null | undefined;
}

export interface TitleProps {
    /**
     * If true, adds padding to the left and right of the title
     */
    addSidePadding?: boolean;
    /** Informational icons displayed to the right of the title */
    adornments?: DisplayAdornment[];
    help?: string;
    /** Icon displayed to the left of the title */
    iconInfo?: IconInfo | null | undefined;
    /** Action icons displayed to the right of the title and adornments */
    options?: {
        iconInfo?: IconInfo | null | undefined;
        label: string;
        onClick: (e?: any) => unknown;
    }[];
    sxs?: {
        stack?: SxType;
        text?: SxType;
    }
    title?: string;
    /** Replaces title if provided */
    titleComponent?: JSX.Element;
    /** Determines size */
    variant?: "header" | "subheader" | "subsection";
}

export interface VersionDisplayProps extends BoxProps {
    confirmVersionChange?: (callback: () => unknown) => unknown;
    currentVersion: Partial<{ versionLabel: string }> | null | undefined;
    loading?: boolean;
    prefix?: string;
    versions?: { versionLabel: string }[];
}
