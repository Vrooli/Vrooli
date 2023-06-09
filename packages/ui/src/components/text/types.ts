import { Api, Organization, Project, Quiz, Routine, SmartContract, Standard, SvgComponent, User } from "@local/shared";
import { BoxProps, TypographyProps } from "@mui/material";
import { ObjectType } from "utils/navigation/openObject";

export interface DateDisplayProps extends BoxProps {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
}

export interface MarkdownDisplayProps {
    content: string | undefined;
    sx?: { [x: string]: any; };
    variant?: TypographyProps["variant"];
}

export interface ObjectTitleProps extends BoxProps {
    language: string;
    languages: string[];
    loading: boolean;
    setLanguage: (language: string) => void;
    translations: { language: string }[];
    title: string | undefined;
    zIndex: number;
}

export interface OwnerLabelProps {
    confirmOpen?: (callback: () => void) => void;
    language?: string
    objectType: ObjectType;
    owner: Routine["owner"] | null | undefined
    sxs?: {
        label?: { [x: string]: any };
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
    /**
     * Icon displayed to the left of the title
     */
    Icon?: SvgComponent;
    options?: {
        Icon: SvgComponent;
        label: string;
        onClick: (e?: any) => void;
    }[];
    sxs?: {
        stack?: { [x: string]: any; };
        text?: { [x: string]: any; };
    }
    title?: string;
    variant: "header" | "subheader";
}

export interface VersionDisplayProps extends BoxProps {
    confirmVersionChange?: (callback: () => void) => void;
    currentVersion: { versionLabel: string } | null | undefined;
    loading?: boolean;
    prefix?: string;
    versions?: { versionLabel: string }[];
}

export interface ViewsDisplayProps {
    views: number | null;
}
