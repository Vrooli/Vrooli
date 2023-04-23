import { partialToString } from "./partialToString";
export const toMutation = async (endpointName, inputType, partial, selectionType) => {
    return await partialToString({
        endpointType: "mutation",
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType,
    });
};
//# sourceMappingURL=toMutation.js.map