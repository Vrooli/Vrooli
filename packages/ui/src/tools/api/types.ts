import { MaybeLazy, NonMaybe } from "../../types";

/**
 * A nested Partial type, where each non-object field is a boolean.
 * Arrays are also treated as objects. Also adds:
 * - The __define field, which can be used to define fragments to include in the selection.
 * - The __union field, which can be used to define a union type (supports using fragments from the __define field)
 * - The __use field, which can be used to reference a single fragment from the __define field 
 */
export type DeepPartialBooleanWithFragments<T extends { __typename: string }> = {
    /**
     * Specifies the selection type
     */
    __selectionType?: 'common' | 'full' | 'list' | 'nav';
    /**
     * Specifies the object type
     */
    __typename?: T['__typename'];
    /**
     * Fragments to include in the selection. Each fragment's key can be used to reference it in the selection.
     */
    __define?: { [key: string]: MaybeLazy<DeepPartialBooleanWithFragments<T>> };
    /**
     * Creates a union of the specified types
     */
    __union?: { [key in T['__typename']]?: (string | number | MaybeLazy<DeepPartialBooleanWithFragments<T>>) };
    /**
     * Defines a fragment to include in the selection. The fragment can be referenced in the selection using the __use field.
     */
    __use?: string | number
} & {
        [P in keyof T]?: T[P] extends Array<infer U> ?
        U extends { __typename: string } ?
        MaybeLazy<DeepPartialBooleanWithFragments<NonMaybe<U>>> :
        boolean :
        T[P] extends { __typename: string } ?
        MaybeLazy<DeepPartialBooleanWithFragments<NonMaybe<T[P]>>> :
        boolean;
    }

/**
 * Ensures that a GraphQL selection is valid for a given type.
 */
export type GqlPartial<
    T extends { __typename: string },
> = {
    /**
     * Must specify the typename, in case we need to use it in a fragment.
     */
    __typename: T['__typename'];
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

}