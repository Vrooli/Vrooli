// Dot Notation
type Prev = [never, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14];
type DottablePaths<T, P extends Prev[number] = 5> = [] | ([P] extends [never] ? never :
    T extends readonly any[] ? never :
    T extends object ? {
        [K in ExtractDottable<keyof T>]: [K, ...DottablePaths<T[K], Prev[P]>]
    }[ExtractDottable<keyof T>] : never);
type Digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9'
type BadChars = '~' | '`' | '!' | '@' | '#' | '%' | '^' | '&' | '*' | '(' | ')' | '-' | '+'
    | '=' | '{' | '}' | ';' | ':' | '\'' | '"' | '<' | '>' | ',' | '.' | '/' | '?'
type ExtractDottable<K extends PropertyKey> =
    K extends string ? string extends K ? never :
    K extends `${Digit}${infer _}` | `${infer _}${BadChars}${infer _}` ? never :
    K
    : never
type Join<T extends string[], D extends string> =
    T extends [] ? never :
    T extends [infer F] ? F :
    T extends [infer F, ...infer R] ?
    F extends string ? string extends F ? string : `${F}${D}${Join<Extract<R, string[]>, D>}` : never : string;
/**
 * Type-safe dot notation for objects
 */
export type DotNotation<T> = Join<Extract<DottablePaths<T, 3>, string[]>, ".">;

/**
 * Prepends a string to all keys in an object
 */
export type PrependString<T extends Record<string, any>, Prefix extends string> = {
    [K in keyof T as `${Prefix}${K & string}`]: T[K]
}

/**
 * A nested Partial type, where each non-object field is a boolean.
 * Arrays are also treated as objects.
 */
export type DeepPartialBoolean<T> = {
    [P in keyof T]?: T[P] extends Array<infer U> ?
    DeepPartialBoolean<U> :
    T[P] extends object ?
    DeepPartialBoolean<T[P]> :
    boolean;
};