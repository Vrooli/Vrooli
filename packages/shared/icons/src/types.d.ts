declare module '@shared/route';
export * from '.';

export interface SvgProps extends React.SVGProps<SVGSVGElement> {
    iconTitle?: string;
    id?: string;
    style?: any;
    onClick?: () => any;
    width?: number | string;
    height?: number | string;
}