export interface SvgProps extends React.SVGProps<SVGSVGElement> {
    iconTitle?: string;
    style?: any;
    onClick?: () => any;
    width?: number | string;
    height?: number | string;
}