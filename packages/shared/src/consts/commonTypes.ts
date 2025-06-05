/* c8 ignore start */
// Dot Notation
// eslint-disable-next-line no-magic-numbers
type Prev = [never, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14];
// eslint-disable-next-line no-magic-numbers
type DottablePaths<T, P extends Prev[number] = 5> = [] | ([P] extends [never] ? never :
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    T extends readonly any[] ? never :
    T extends object ? {
        [K in ExtractDottable<keyof T>]: [K, ...DottablePaths<T[K], Prev[P]>]
    }[ExtractDottable<keyof T>] : never);
type Digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
type BadChars = "~" | "`" | "!" | "@" | "#" | "%" | "^" | "&" | "*" | "(" | ")" | "-" | "+"
    | "=" | "{" | "}" | ";" | ":" | "'" | "\"" | "<" | ">" | "," | "." | "/" | "?"
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
// eslint-disable-next-line @typescript-eslint/no-explicit-any
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type PassableLoggerData = Record<string, any>;
/**
 * Logger that is passed into a function. Typically `console` in the UI 
 * and winston in the server.
 */
export type PassableLogger = {
    error: (message: string, data?: PassableLoggerData) => unknown;
    info: (message: string, data?: PassableLoggerData) => unknown;
    warn: (message: string, data?: PassableLoggerData) => unknown;
}

/**
 * Converts objects as represented in the UI (especially forms) to create/update 
 * input objects for the GraphQL API.
 */
export type ShapeModel<
    T extends object,
    TCreate extends object | null,
    TUpdate extends object | null
> = (TCreate extends null ? object : { create: (item: T) => TCreate }) &
    (TUpdate extends null ? object : {
        update: (o: T, u: T) => TUpdate | undefined,
        hasObjectChanged?: (o: T, u: T) => boolean,
    }) & { idField?: keyof T & string }

export type CanConnect<
    RelationShape extends ({ [key in IDField]: string } & { __typename: string }),
    IDField extends string = "id",
> = RelationShape | (Pick<RelationShape, IDField | "__typename"> & { __connect?: boolean } & { [key: string]: any });


/**
 * A general status state for an object
 */
export enum Status {
    /**
     * Routine would be valid, except there are unlinked nodes
     */
    Incomplete = "Incomplete",
    /**
     * Something is wrong with the routine (e.g. no end node)
     */
    Invalid = "Invalid",
    /**
     * The routine is valid, and all nodes are linked
     */
    Valid = "Valid",
}

export type ValueOf<T> = T[keyof T];
