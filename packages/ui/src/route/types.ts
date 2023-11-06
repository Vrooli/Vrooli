export type SetLocationOptions = {
    replace?: boolean;
    searchParams?: Record<string, any>;
    /** True if navigation-related confirmation prompts (e.g. dirty form) should be ignored */
    bypassBlock?: boolean;
};
export type SetLocation = (to: string, options?: SetLocationOptions) => unknown;
