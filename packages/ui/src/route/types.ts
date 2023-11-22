export type SetLocationOptions = {
    replace?: boolean;
    searchParams?: Record<string, any>;
};
export type SetLocation = (to: string, options?: SetLocationOptions) => unknown;
