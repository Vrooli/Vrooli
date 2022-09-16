import { BoxProps, TypographyProps } from "@mui/material";
import { Session } from "types";
import { ObjectType } from "utils";

export interface DateDisplayProps extends BoxProps {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
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

export interface TextShrinkProps extends TypographyProps {
    id: string;
    minFontSize?: string | number;
}

export interface VersionDisplayProps extends BoxProps {
    handleVersionChange: (version: number) => void;
    loading?: boolean;
    showIcon?: boolean;
    versions?: string[];
}