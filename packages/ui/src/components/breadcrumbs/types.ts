import { SxType } from "types";

export interface BreadcrumbsBaseProps {
    ariaLabel?: string;
    onClick?: (e: React.MouseEvent<HTMLAnchorElement>) => unknown;
    paths: readonly { text: string; link: string; }[];
    separator?: string;
    sx?: SxType;
    textColor?: string;
}

export type CopyrightBreadcrumbsProps = Omit<BreadcrumbsBaseProps, "paths" | "ariaLabel">

export type PolicyBreadcrumbsProps = Omit<BreadcrumbsBaseProps, "paths" | "ariaLabel">
