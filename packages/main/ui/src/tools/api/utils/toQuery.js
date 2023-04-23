import { partialToString } from "./partialToString";
export const toQuery = async (endpointName, inputType, partial, selectionType) => {
    return await partialToString({
        endpointType: "query",
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType,
    });
};
//# sourceMappingURL=toQuery.js.map