import { SxType } from "types";

export interface BreadcrumbsBaseProps {
    paths: { text: string; link: string; }[];
    separator?: string;
    ariaLabel?: string;
    textColor?: string;
    sx?: SxType;
}

export type CopyrightBreadcrumbsProps = Omit<BreadcrumbsBaseProps, "paths" | "ariaLabel">

export type PolicyBreadcrumbsProps = Omit<BreadcrumbsBaseProps, "paths" | "ariaLabel">
