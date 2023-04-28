import { SvgProps } from "@local/shared";

export interface DiagonalWaveLoaderProps {
    size?: number;
    color?: string;
    sx?: { [key: string]: any };
}

export type PageTab<T> = {
    color?: string,
    href?: string,
    /**
     * If set, icon is displayed and label becomes a toolip
     */
    Icon?: (props: SvgProps) => JSX.Element,
    index: number,
    label: string,
    value: T
};

export interface PageTabsProps<T> {
    ariaLabel: string,
    currTab: PageTab<T>,
    fullWidth?: boolean,
    onChange: (event: React.ChangeEvent<unknown>, value: any) => void,
    tabs: PageTab<T>[],
}

export interface TwinklingStarsProps {
    amount?: number;
    size?: number;
    color?: string;
    speed?: number;
    sx?: { [key: string]: any };
}
