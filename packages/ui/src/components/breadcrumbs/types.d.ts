export interface BreadcrumbsBaseProps {
    paths: { text: string; link: string; }[];
    separator?: string;
    ariaLabel?: string;
    textColor?: string;
    style?: object;
    className?: string;
}