import { BoxProps, TypographyProps } from "@mui/material";
import { Api, Organization, Project, Quiz, Routine, Session, SmartContract, Standard, User } from "@shared/consts";
import { CommonKey } from "@shared/translations";
import { VersionInfo } from "types";
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
    owner: Routine['owner'] | null | undefined
    session: Session;
    sxs?: {
        label?: { [x: string]: any };
    }
}

export interface PageTitleProps {
    helpKey?: CommonKey;
    helpVariables?: { [x: string]: string | number };
    titleKey: CommonKey;
    titleVariables?: { [x: string]: string | number };
    session: Session;
    sxs?: { 
        stack?: { [x: string]: any; };
        text?: { [x: string]: any; };
    }
}

export type StatsCompactPropsObject = Api | Organization | Project | Quiz | Routine | SmartContract | Standard | User;
export interface StatsCompactProps<T extends StatsCompactPropsObject> {
    handleObjectUpdate: (object: T) => void;
    loading: boolean;
    object: T | null | undefined;
    session: Session;
}

export interface TextShrinkProps extends TypographyProps {
    id: string;
    minFontSize?: string | number;
}

export interface VersionDisplayProps extends BoxProps {
    confirmVersionChange?: (callback: () => void) => void;
    currentVersion: VersionInfo | null | undefined;
    loading?: boolean;
    prefix?: string;
    versions?: VersionInfo[];
}

export interface ViewsDisplayProps {
    views: number | null;
}