/** Makes a value nullable. Mimics the Maybe type in GraphQL. */
type Maybe<T> = T | null;

/** Recursively removes the Maybe type from all fields in a type, and makes them required. */
export type NonMaybe<T> = { [K in keyof T]-?: T[K] extends Maybe<unknown> ? NonNullable<T[K]> : T[K] };

/** Makes a value lazy or not */
export type MaybeLazyAsync<T> = T | (() => T) | (() => Promise<T>);

/**
 * A nested Partial type, where each non-object field is a boolean.
 * Arrays are also treated as objects. Also adds:
 * - The __union field, which represents a union type
 */
export type DeepPartialBooleanWithFragments<T extends object> = {
    /**
     * Creates a union of the specified types
     */
    __union?: Record<string, (string | number | MaybeLazyAsync<DeepPartialBooleanWithFragments<T>>)>;
} & {
        [P in keyof T]?: T[P] extends Array<infer U> ?
        U extends object ?
        MaybeLazyAsync<DeepPartialBooleanWithFragments<NonMaybe<U>>> :
        boolean :
        T[P] extends object ?
        MaybeLazyAsync<DeepPartialBooleanWithFragments<NonMaybe<T[P]>>> :
        boolean;
    }

/**
 * Ensures that a GraphQL selection is valid for a given type.
 */
export type GqlPartial<
    T extends object,
> = {
    /**
     * Fields which are always included. This is is recursive partial of T
     */
    common?: DeepPartialBooleanWithFragments<NonMaybe<T>>;
    /**
     * Fields included in the full selection. Combined with common.
     */
    full?: DeepPartialBooleanWithFragments<NonMaybe<T>>;
    /**
     * Fields included in the minimal (list) selection. Combined with common.
     * If not provided, defaults to the same as full.
     */
    list?: DeepPartialBooleanWithFragments<NonMaybe<T>>;
    /**
     * Fields included to get the name and navigation info for an object.
     * NOT combined with common.
     */
    nav?: DeepPartialBooleanWithFragments<NonMaybe<T>>;

};

export type SelectionType = "common" | "full" | "list" | "nav";
