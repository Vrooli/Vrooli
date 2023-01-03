/**
 * Helper function for formatting an object for an update mutation
 * @param updated The updated object
 * @param shape Shapes the updated object for the update mutation
 */
export const shapeUpdate = <
    Input extends {},
    Output extends {}
>(
    updated: Input | null | undefined,
    shape: Output | (() => Output),
): Output | undefined => {
    if (!updated) return undefined;
    let result = typeof shape === 'function' ? (shape as () => Output)() : shape;
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as Output;
    // Return result if it is not empty
    return result && Object.keys(result).length > 0 ? result : undefined;
}