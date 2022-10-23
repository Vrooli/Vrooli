declare module '@shared/route';
export * from '.';

export interface SvgProps {
    fill?: string;
    iconTitle?: string;
    id?: string;
    style?: any;
    onClick?: () => any;
    width?: number | string | null;
    height?: number | string | null;
}

export type SvgComponent = (props: SvgProps) => JSX.Element;