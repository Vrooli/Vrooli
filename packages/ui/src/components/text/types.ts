import { Routine } from "@local/shared";
import { BoxProps, TypographyProps } from "@mui/material";
import { SvgComponent, SxType } from "types";
import { ListObject } from "utils/display/listTools";
import { ObjectType } from "utils/navigation/openObject";

export interface DateDisplayProps extends Omit<BoxProps, "zIndex"> {
    loading?: boolean;
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

export interface TextShrinkProps extends TypographyProps {
    id: string;
    minFontSize?: string | number;
}

export interface TitleProps {
    /** Informational icons displayed to the right of the title */
    adornments?: JSX.Element[];
    help?: string;
    /** Icon displayed to the left of the title */
    Icon?: SvgComponent;
    /** Action icons displayed to the right of the title and adornments */
    options?: {
        Icon: SvgComponent;
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
    variant?: "header" | "subheader";
}

export interface VersionDisplayProps extends BoxProps {
    confirmVersionChange?: (callback: () => unknown) => unknown;
    currentVersion: Partial<{ versionLabel: string }> | null | undefined;
    loading?: boolean;
    prefix?: string;
    versions?: { versionLabel: string }[];
}
