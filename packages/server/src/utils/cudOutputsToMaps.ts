import { type ModelType } from "@vrooli/shared";
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
            const node = inputsById[createdId].node;
            if (node.action !== "Create") {
                // If this error is thrown, there is a bug in cudInputsToMaps
                throw new CustomError("0529", "InternalError", { node, createdId });
            }
            const type = node.__typename;
            initResult(type as ModelType);
            result[type]!.createInputs.push(inputsById[createdId].input as Model["ApiCreate"]);
        }
    }
    // Generate updateInputs
    if (idsByAction.Update) {
        for (const updatedId of idsByAction.Update) {
            const node = inputsById[updatedId].node;
            if (node.action !== "Update") {
                // If this error is thrown, there is a bug in cudInputsToMaps
                throw new CustomError("0530", "InternalError", { node, updatedId });
            }
            const type = node.__typename;
            initResult(type as ModelType);
            // Populate the update input for the ID
            result[type]!.updateInputs.push(inputsById[updatedId].input as Model["ApiUpdate"]);
        }
    }
    // Generate createdIds and updatedIds
    for (const type of Object.keys(result)) {
        const { idField } = ModelMap.getLogic(["idField"], type as ModelType);
        for (const createInput of result[type].createInputs) {
            result[type].createdIds.push((createInput as { [key in typeof idField]: string })[idField]);
        }
        for (const updateInput of result[type].updateInputs) {
            result[type].updatedIds.push((updateInput as { [key in typeof idField]: string })[idField]);
        }
    }
    return result;
}
