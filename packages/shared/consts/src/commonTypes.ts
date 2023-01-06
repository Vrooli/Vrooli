type Dec = [never, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
type PathsToStringProps<T, MaxDepth extends number> = MaxDepth extends 0 ? [] :
    T extends string ? [] : {
        [K in Extract<keyof T, string>]: [K, ...PathsToStringProps<T[K], Dec[MaxDepth]>]
    }[Extract<keyof T, string>];
type Join<T extends string[], D extends string> =
    T extends [] ? never :
    T extends [infer F] ? F :
    T extends [infer F, ...infer R] ?
    F extends string ?
    `${F}${D}${Join<Extract<R, string[]>, D>}` : never : string;
/**
 * Type-safe dot notation for an object
 */
export type DotNotation<T extends {}, MaxDepth extends number> = Join<PathsToStringProps<T, MaxDepth>, '.'>;

/**
 * Prepends a string to all keys in an object
 */
export type PrependString<T extends Record<string, any>, Prefix extends string> = {
    [K in keyof T as `${Prefix}${K & string}`]: T[K]
}