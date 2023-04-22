/**
 * Checks if an obect: 
 * 1. Exists
 * 2. Has the "__typename" property
 * 3. The "__typename" property is one of the provided types
 * @param obj The object to check
 * @param types The types to check against
 * @returns True if the object is of one of the provided types, false otherwise. 
 * The function is type safe, so code called after this function will be aware of the type of the object.
 */
export const isOfType = <T extends string>(obj: any, ...types: T[]): obj is { __typename: T } => {
    if (!obj || !obj.__typename) return false;
    return types.includes(obj.__typename);
};
