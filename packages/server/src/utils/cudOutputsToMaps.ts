// AI_CHECK: TYPE_SAFETY=phase2-outputs | LAST: 2025-07-04 - Enhanced type safety with proper type assertions and guards
import { type ModelType } from "@vrooli/shared";
import { hasTypename } from "./typeGuards.js";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { type ModelLogicType } from "../models/types.js";
import { type IdsByAction, type InputsById } from "./types.js";

type CudOutputData<Model extends {
    ApiCreate: ModelLogicType["ApiCreate"],
    ApiUpdate: ModelLogicType["ApiUpdate"],
    ApiModel: ModelLogicType["ApiModel"],
}> = {
    createdIds: string[],
    createInputs: Model["ApiCreate"][],
    updatedIds: string[],
    updateInputs: Model["ApiUpdate"][],
}

export function cudOutputsToMaps<Model extends {
    ApiCreate: ModelLogicType["ApiCreate"],
    ApiUpdate: ModelLogicType["ApiUpdate"],
    ApiModel: ModelLogicType["ApiModel"],
}>({
    idsByAction,
    inputsById,
}: {
    idsByAction: IdsByAction,
    inputsById: InputsById,
}): { [key in `${ModelType}`]?: CudOutputData<Model> } {
    const result: { [key in `${ModelType}`]?: CudOutputData<Model> } = {};
    // Helper function to initialize the result object for a given type
    function initResult(type: `${ModelType}`) {
        if (!result[type]) {
            result[type] = {
                createdIds: [],
                createInputs: [],
                updatedIds: [],
                updateInputs: [],
            };
        }
    }
    // Generate createInputs
    if (idsByAction.Create) {
        for (const createdId of idsByAction.Create) {
            const node = inputsById[createdId.toString()].node;
            if (node.action !== "Create") {
                // If this error is thrown, there is a bug in cudInputsToMaps
                throw new CustomError("0529", "InternalError", { node, createdId });
            }
            if (!hasTypename(node)) {
                throw new CustomError("0531", "InternalError", { node, createdId });
            }
            const type = node.__typename;
            initResult(type);
            result[type]!.createInputs.push(inputsById[createdId.toString()].input as Model["ApiCreate"]);
        }
    }
    // Generate updateInputs
    if (idsByAction.Update) {
        for (const updatedId of idsByAction.Update) {
            const node = inputsById[updatedId.toString()].node;
            if (node.action !== "Update") {
                // If this error is thrown, there is a bug in cudInputsToMaps
                throw new CustomError("0530", "InternalError", { node, updatedId });
            }
            if (!hasTypename(node)) {
                throw new CustomError("0532", "InternalError", { node, updatedId });
            }
            const type = node.__typename;
            initResult(type);
            // Populate the update input for the ID
            result[type]!.updateInputs.push(inputsById[updatedId.toString()].input as Model["ApiUpdate"]);
        }
    }
    // Generate createdIds and updatedIds
    for (const type of Object.keys(result)) {
        const typeKey = type as ModelType;
        const { idField } = ModelMap.getLogic(["idField"], typeKey);
        const typeResult = result[typeKey];
        if (!typeResult) continue;
        
        for (const createInput of typeResult.createInputs) {
            const createInputWithId = createInput as { [key in typeof idField]: string };
            typeResult.createdIds.push(createInputWithId[idField]);
        }
        for (const updateInput of typeResult.updateInputs) {
            const updateInputWithId = updateInput as { [key in typeof idField]: string };
            typeResult.updatedIds.push(updateInputWithId[idField]);
        }
    }
    return result;
}
