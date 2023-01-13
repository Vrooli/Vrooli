import { SnackSeverity } from "components";
import { PubSub } from "utils/pubsub";

/**
 * Helper function for formatting an object for an update mutation
 * @param updated The updated object
 * @param shape Shapes the updated object for the update mutation
 * @param assertHasUpdate Asserts that the updated object has at least one value
 */
export const shapeUpdate = <
    Input extends {},
    Output extends {},
    AssertHasUpdate extends boolean = false
>(
    updated: Input | null | undefined,
    shape: Output | (() => Output),
    assertHasUpdate: AssertHasUpdate = false as AssertHasUpdate
): Output | undefined => {
    if (!updated) return undefined;
    let result = typeof shape === 'function' ? (shape as () => Output)() : shape;
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as Output;
    // If assertHasUpdate is true, make sure that the result has at least one value
    if (assertHasUpdate && (!result || Object.keys(result).length === 0)) {
        PubSub.get().publishSnack({ messageKey: 'NothingToUpdate', severity: SnackSeverity.Error });
        return undefined;
    }
    return result
}