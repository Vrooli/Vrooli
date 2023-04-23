import { PubSub } from "../../../pubsub";
export const shapeUpdate = (updated, shape, assertHasUpdate = false) => {
    if (!updated)
        return undefined;
    let result = typeof shape === "function" ? shape() : shape;
    if (result)
        result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined));
    if (assertHasUpdate && (!result || Object.keys(result).length === 0)) {
        PubSub.get().publishSnack({ messageKey: "NothingToUpdate", severity: "Error" });
        return undefined;
    }
    return result;
};
//# sourceMappingURL=shapeUpdate.js.map