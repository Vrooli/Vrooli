import { BoxProps, TypographyProps } from "@mui/material";
import { Project, Routine, Session, Standard } from "types";
import { ObjectType } from "utils";

export interface DateDisplayProps extends BoxProps {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
}

export interface ObjectTitleProps extends BoxProps {
    language: string;
    loading: boolean;
    session: Session;
    setLanguage: (language: string) => void;
    translations: { language: string }[];
    title: string | undefined;
    zIndex: number;
}

export interface OwnerLabelProps {
    confirmOpen?: (callback: () => void) => void;
    language?: string
    objectType: ObjectType;
    owner: Project['owner'] | Routine['owner'] | Standard['creator'] | null | undefined
    session: Session;
    sxs?: {
        label?: { [x: string]: any };
    }
}

export interface PageTitleProps {
    helpText?: string;
    title: string;
    sxs?: { 
        stack?: { [x: string]: any; };
        text?: { [x: string]: any; };
    }
}

export type StatsCompactPropsObject = Project | Routine | Standard;
export interface StatsCompactProps<T extends StatsCompactPropsObject> {
    handleObjectUpdate: (object: T) => void;
    loading: boolean;
    object: T | null;
    session: Session;
}

export interface TextShrinkProps extends TypographyProps {
    id: string;
    minFontSize?: string | number;
}

export interface VersionDisplayProps extends BoxProps {
    confirmVersionChange?: (callback: () => void) => void;
    currentVersion: string | null | undefined;
    loading?: boolean;
    prefix?: string;
    versions?: string[];
}

export interface ViewsDisplayProps {
    views: number | null;
}