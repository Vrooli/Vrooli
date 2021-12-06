export interface SvgProps extends React.SVGProps<SVGSVGElement> {
    iconTitle?: string;
    className?: string;
    onClick?: () => any;
    width?: number | string;
    height?: number | string;
}