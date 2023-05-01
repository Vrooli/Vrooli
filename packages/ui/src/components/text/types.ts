import { Api, Organization, Project, Quiz, Routine, SmartContract, Standard, SvgComponent, User } from "@local/shared";
import { BoxProps, TypographyProps } from "@mui/material";
import { ObjectType } from "utils/navigation/openObject";

export interface DateDisplayProps extends BoxProps {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
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
    owner: Routine['owner'] | null | undefined
    sxs?: {
        label?: { [x: string]: any };
    }
}

export interface HeaderProps {
    help?: string | undefined;
    sxs?: {
        stack?: { [x: string]: any; };
        text?: { [x: string]: any; };
    }
    title: string | undefined;
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

export interface SubheaderProps {
    help?: string | undefined;
    Icon?: SvgComponent;
    sxs?: {
        stack?: { [x: string]: any; };
        text?: { [x: string]: any; };
    }
    title: string | undefined;
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