export type PageTab<T extends any> = {
    color?: string,
    href?: string,
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