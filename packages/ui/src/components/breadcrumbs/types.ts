import { CommonProps } from "types";

export interface BreadcrumbsBaseProps {
    paths: { text: string; link: string; }[];
    separator?: string;
    ariaLabel?: string;
    textColor?: string;
    sx?: any;
}

export type CopyrightBreadcrumbsProps = Pick<CommonProps, 'session'> & Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>

export type PolicyBreadcrumbsProps = Pick<CommonProps, 'session'> & Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>