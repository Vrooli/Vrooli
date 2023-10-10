import { GqlModelType } from "@local/shared";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { ModelLogicType } from "../models/types";
import { IdsByAction, InputsById } from "./types";

type CudOutputData<Model extends {
    GqlCreate: ModelLogicType["GqlCreate"],
    GqlUpdate: ModelLogicType["GqlUpdate"],
    GqlModel: ModelLogicType["GqlModel"],
}> = {
    createdIds: string[],
    createInputs: Model["GqlCreate"][],
    updatedIds: string[],
    updateInputs: Model["GqlUpdate"][],
}

export const cudOutputsToMaps = <Model extends {
    GqlCreate: ModelLogicType["GqlCreate"],
    GqlUpdate: ModelLogicType["GqlUpdate"],
    GqlModel: ModelLogicType["GqlModel"],
}>({
    idsByAction,
    inputsById,
}: {
    idsByAction: IdsByAction,
    inputsById: InputsById,
}): { [key in `${GqlModelType}`]?: CudOutputData<Model> } => {
    const result: { [key in `${GqlModelType}`]?: CudOutputData<Model> } = {};
    // Helper function to initialize the result object for a given type
    const initResult = (type: `${GqlModelType}`) => {
        if (!result[type]) {
            result[type] = {
                createdIds: [],
                createInputs: [],
                updatedIds: [],
                updateInputs: [],
            };
        }
    };
    // Generate createInputs
    if (idsByAction.Create) {
        for (const createdId of idsByAction.Create) {
            const node = inputsById[createdId].node;
            if (node.action !== "Create") {
                // If this error is thrown, there is a bug in cudInputsToMaps
                throw new CustomError("0529", "InternalError", ["en"], { node, createdId });
            }
            const type = node.__typename;
            initResult(type as GqlModelType);
            result[type]!.createInputs.push(inputsById[createdId].input as Model["GqlCreate"]);
        }
    }
    // Generate updateInputs
    if (idsByAction.Update) {
        for (const updatedId of idsByAction.Update) {
            const node = inputsById[updatedId].node;
            if (node.action !== "Update") {
                // If this error is thrown, there is a bug in cudInputsToMaps
                throw new CustomError("0530", "InternalError", ["en"], { node, updatedId });
            }
            const type = node.__typename;
            initResult(type as GqlModelType);
            // Populate the update input for the ID
            result[type]!.updateInputs.push(inputsById[updatedId].input as Model["GqlUpdate"]);
        }
    }
    // Generate createdIds and updatedIds
    for (const type of Object.keys(result) as GqlModelType[]) {
        const { idField } = ModelMap.getLogic(["idField"], type);
        for (const createInput of result[type]!.createInputs) {
            result[type]!.createdIds.push((createInput as { [key in typeof idField]: string })[idField]);
        }
        for (const updateInput of result[type]!.updateInputs) {
            result[type]!.updatedIds.push((updateInput as { [key in typeof idField]: string })[idField]);
        }
    }
    return result;
};
