export interface BreadcrumbsBaseProps {
    paths: { text: string; link: string; }[];
    separator?: string;
    ariaLabel?: string;
    textColor?: string;
    style?: object;
    className?: string;
}

export type CopyrightBreadcrumbsProps = Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel' | 'style'>

export type PolicyBreadcrumbsProps = Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>