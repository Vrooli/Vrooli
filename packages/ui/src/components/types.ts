import { SvgProps } from "@shared/icons";

export type PageTab<T extends any> = {
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

export interface PageTabsProps<T extends any> {
    ariaLabel: string,
    currTab: PageTab<T>,
    onChange: (event: React.ChangeEvent<{}>, value: any) => void,
    tabs: PageTab<T>[],
}