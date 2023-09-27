import { GqlModelType } from "@local/shared";
import { InputsById } from "./types";

/**
* Helper function to find the parent validation data for an object
* @param id ID of object to find parent for
* @param typename Only return parent if it matches this typename
* @param inputsById Map of all input data, keyed by ID
* @returns Input data for parent object, or undefined if parent doesn't exist or doesn't match typename
*/
export const getParentInfo = (id: string, typename: `${GqlModelType}`, inputsById: InputsById): any | undefined => {
    const node = inputsById[id]?.node;
    if (node?.__typename !== typename) return undefined;
    return node?.parent ? inputsById[node.parent.id]?.input : undefined;
};
