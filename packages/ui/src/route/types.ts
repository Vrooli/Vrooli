// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 'any' type with proper union type for searchParams
export type SetLocationOptions = {
    replace?: boolean;
    searchParams?: Record<string, string | number | boolean | undefined>;
};
export type SetLocation = (to: string, options?: SetLocationOptions) => unknown;
