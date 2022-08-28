import { BoxProps, TypographyProps } from "@mui/material";

export interface DateDisplayProps extends BoxProps {
    loading?: boolean;
    showIcon?: boolean;
    textBeforeDate?: string;
    timestamp?: number;
}

export interface TextShrinkProps extends TypographyProps { 
    id: string;
}