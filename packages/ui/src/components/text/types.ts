import { Api, Organization, Project, Quiz, Routine, SmartContract, Standard, User } from "@local/shared";
import { BoxProps, TypographyProps } from "@mui/material";
import { SvgComponent, SxType } from "types";
import { ObjectType } from "utils/navigation/openObject";

export interface DateDisplayProps extends Omit<BoxProps, "zIndex"> {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
    zIndex: number;
}

export interface MarkdownDisplayProps {
    content: string | undefined;
    isEditable?: boolean;
    onChange?: (content: string) => void;
    sx?: SxType;
    variant?: TypographyProps["variant"];
    zIndex: number;
}

export interface OwnerLabelProps {
    confirmOpen?: (callback: () => void) => void;
    language?: string
    objectType: ObjectType;
    owner: Routine["owner"] | null | undefined
    sxs?: {
        label?: SxType;
    }
}

export type StatsCompactPropsObject = Api | Organization | Project | Quiz | Routine | SmartContract | Standard | User;
export interface StatsCompactProps<T extends StatsCompactPropsObject> {
    handleObjectUpdate: (object: T) => void;
    loading: boolean;
    object: T | null | undefined;
}

export interface TextShrinkProps extends TypographyProps {
    id: string;
    minFontSize?: string | number;
}

export interface TitleProps {
    help?: string;
    /** Icon displayed to the left of the title */
    Icon?: SvgComponent;
    options?: {
        Icon: SvgComponent;
        label: string;
        onClick: (e?: any) => void;
    }[];
    sxs?: {
        stack?: SxType;
        text?: SxType;
    }
    title?: string;
    /** Replaces title if provided */
    titleComponent?: JSX.Element;
    /** Determines size */
    variant: "header" | "subheader";
    zIndex: number;
}

export interface VersionDisplayProps extends BoxProps {
    confirmVersionChange?: (callback: () => void) => void;
    currentVersion: { versionLabel: string } | null | undefined;
    loading?: boolean;
    prefix?: string;
    versions?: { versionLabel: string }[];
    zIndex: number;
}

export interface ViewsDisplayProps {
    views: number | null;
}
