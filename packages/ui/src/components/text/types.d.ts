import { BoxProps, TypographyProps } from "@mui/material";

export interface DateDisplayProps extends BoxProps {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
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