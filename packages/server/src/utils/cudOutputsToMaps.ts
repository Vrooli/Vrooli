import { Count, GqlModelType } from "@local/shared";
import { ModelLogicType } from "../models/types";
import { RecursivePartial } from "../types";
import { IdsByAction, IdsByType, InputsById } from "./types";

type CudOutputData<Model extends {
    GqlUpdate: ModelLogicType["GqlUpdate"],
    GqlModel: ModelLogicType["GqlModel"],
}> = {
    created: (RecursivePartial<Model["GqlModel"]> & { id: string })[],
    updated: (RecursivePartial<Model["GqlModel"]> & { id: string })[],
    deleted: Count,
    deletedIds: string[],
    updateInputs: Model["GqlUpdate"][],
}

export const cudOutputsToMaps = async <Model extends {
    GqlUpdate: ModelLogicType["GqlUpdate"],
    GqlModel: ModelLogicType["GqlModel"],
}>({
    idsByAction,
    idsByType,
    inputsById,
    topLevelResults,
}: {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsById: InputsById,
    topLevelResults: { __typename: `${GqlModelType}`, action: "Create" | "Update", result: object }[],
}): Promise<{ [key in `${GqlModelType}`]?: CudOutputData<Model> }> => {
    const result: { [key in `${GqlModelType}`]?: CudOutputData<Model> } = {};
    // Helper function to initialize the result object for a given type
    const initResult = (type: `${GqlModelType}`) => {
        if (!result[type]) {
            result[type] = {
                created: [],
                updated: [],
                deleted: { __typename: "Count" as const, count: 0 },
                deletedIds: [],
                updateInputs: [],
            };
        }
    };
    // If there are update actions, generate the update inputs TODO make sure that each input has a node. Not sure if this is always the case, but it should be
    if (idsByAction.Update) {
        for (const updatedId of idsByAction.Update) {
            const node = inputsById[updatedId].node;
            // Check if the action is "Update" (although this might be redundant if we trust idsByAction)
            if (node.action === "Update") {
                const type = node.__typename;
                initResult(type as GqlModelType);
                // Populate the update input for the ID
                result[type]!.updateInputs.push(inputsById[updatedId].input as Model["GqlUpdate"]);
            }
        }
    }
    // Loop through the top level results
    for (const { __typename, action, result } of topLevelResults) {
        initResult(__typename);
        // If the action is "Create", add the output to the created array
        if (action === "Create") {
            result[__typename]!.created.push(result);
        }
        // If the action is "Update", add the output to the updated array
        if (action === "Update") {
            result[__typename]!.updated.push(result);
        }
        // TODO recurse
    }
    // TODO populate created, updated, deleted, and deletedIds
    return result;
};
