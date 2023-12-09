import { GqlModelType } from "@local/shared";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { Formatter, ModelLogicType } from "../models/types";
import { PrismaType } from "../types";
import { getActionFromFieldName } from "./getActionFromFieldName";
import { CudInputData, IdsByAction, IdsByPlaceholder, IdsByType, IdsCreateToConnect, InputNode, InputsById, InputsByType, QueryAction } from "./types";

/**
 * Converts placeholder ids to actual IDs, or null if actual ID not found. 
 * This is only needed for idsByAction and idsByType, as you can see later 
 * in the code by looking for what the `placeholder` variable is added to.
 */
const convertPlaceholders = async ({
    idsByAction,
    idsByType,
    prisma,
    languages,
}: {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    prisma: PrismaType,
    languages: string[]
}): Promise<void> => {
    // Store processed placeholders in a map to avoid duplicate queries
    const placeholderToIdMap: IdsByPlaceholder = {};

    // Helper function to fetch and map placeholders
    const fetchAndMapPlaceholder = async (placeholder: string): Promise<string | null> => {
        if (placeholder in placeholderToIdMap) {
            return placeholderToIdMap[placeholder];  // Already processed this placeholder
        }

        const [objectType, path] = placeholder.split("|", 2);
        const [rootId, ...relations] = path.split(".");

        const { delegate } = ModelMap.getLogic(["delegate"], objectType as GqlModelType);
        const queryResult = await delegate(prisma).findUnique({
            where: { id: rootId },
            select: relations.reduce((selectObj, relation) => {
                return { [relation]: selectObj };
            }, {}),
        });

        let currentObject = queryResult;
        for (const relation of relations) {
            currentObject = currentObject![relation];
        }

        const resultId = currentObject && currentObject.id ? currentObject.id : null;
        placeholderToIdMap[placeholder] = resultId;
        return resultId;
    };

    // Placeholders should be in both objects, so we only need to iterate through one of them
    for (const [type, ids] of Object.entries(idsByType)) {
        const placeholders = ids.filter(id => typeof id === "string" && id.includes("|"));
        if (placeholders.length === 0) continue;
        for (const placeholder of placeholders) {
            const existingId = await fetchAndMapPlaceholder(placeholder);
            // Replace in idsByType
            idsByType[type] = ids.map(id => id === placeholder ? existingId : id);
            // Find action and index in action array
            // type IdsByAction = { [action in QueryAction]?: string[] };
            const action = Object.entries(idsByAction).find(([, ids]) => ids.includes(placeholder));
            if (!action) continue;
            const [actionType, actionIds] = action;
            const index = actionIds.indexOf(placeholder);
            // Replace in idsByAction
            idsByAction[actionType] = actionIds.map((id, i) => i === index ? existingId : id);
        }
    }
};

// TODO Try adding `readMany` so we can use this for reads. I believe the current way we do reads doesn't properly check relation permissions, so this would solve that.
export const cudInputsToMaps = async ({
    inputData,
    prisma,
    languages,
}: {
    inputData: CudInputData[],
    prisma: PrismaType,
    languages: string[]
}): Promise<{
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    idsCreateToConnect: IdsCreateToConnect,
    inputsById: InputsById,
    inputsByType: InputsByType,
}> => {
    const idsByAction: IdsByAction = {};
    const idsByType: IdsByType = {};
    const idsCreateToConnect: IdsCreateToConnect = {};
    const inputsById: InputsById = {};
    const inputsByType: InputsByType = {};

    const initByAction = (actionType: QueryAction) => {
        if (!idsByAction[actionType]) {
            idsByAction[actionType] = [];
        }
    };
    const initByType = (objectType: `${GqlModelType}`) => {
        if (!idsByType[objectType]) {
            idsByType[objectType] = [];
        }
        if (!inputsByType[objectType]) {
            inputsByType[objectType] = {
                Connect: [],
                Create: [],
                Delete: [],
                Disconnect: [],
                Read: [],
                Update: [],
            };
        }
    };

    /**
     * Recursively builds a tree from a mutation input 
     * to optimize the performance of various operations performed before and after creating, 
     * updating, deleting, connecting, and disconnecting objects.
     *
     * NOTE: For operations like "Connect" or "Disconnect" on non-array relations (i.e. one-to-one or many-to-one), 
     * we may be implicitly connecting/disconnecting an object without knowing its ID. In this case, we generate
     * a placeholder ID that encodes all the information needed to query the database for the actual ID.
     *
     * @param actionType - The type of mutation action. Example: "Create" 
     * @param input - The GraphQL mutation input object
     * @param format - ModelLogic property which contains formatting information for the current object type
     * @param languages - The languages to use for error messages
     * @param idField - The name of the ID field for the current object type. Defaults to "id".
     * @param closestWithId - Keeps track of the closest known object ID as we traverse the input. Useful 
     *                             for generating placeholders.
     * @returns rootNode - The root of the hierarchical tree representation of the input.
     */
    const buildTree = <
        Typename extends `${GqlModelType}`,
        GqlCreate extends ModelLogicType["GqlCreate"],
        GqlModel extends ModelLogicType["GqlModel"],
        PrismaModel extends ModelLogicType["PrismaModel"],
    >(
        actionType: QueryAction,
        input: GqlModel,
        format: Formatter<{ __typename: Typename, GqlCreate: GqlCreate, GqlModel: GqlModel, PrismaModel: PrismaModel }>,
        languages: string[],
        idField = "id",
        closestWithId: { __typename: string, id: string, path: string } | null = { __typename: "", id: "", path: "" },
    ): InputNode => {
        // Initialize tree
        const rootNode = new InputNode(format.gqlRelMap.__typename, input[idField], actionType);
        // Initialize maps
        initByAction(actionType);
        initByType(format.gqlRelMap.__typename);
        idsByAction[actionType]?.push(input[idField]);
        idsByType[format.gqlRelMap.__typename]?.push(input[idField]);
        // Initialize object that we'll push to inputsByType later
        const inputInfo: { node: InputNode, input: any } = { node: rootNode, input: {} };

        // Update the closestId when a new ID is found, for use in generating placeholders 
        // The only exception is when a "Create" action is found, as we know that the object doesn't exist yet, 
        // and can thus ignore any following "Connect" or "Disconnect" actions.
        if (actionType === "Create") {
            closestWithId = null; // Set to null to ignore any following "Connect" or "Disconnect" actions
        } else if (closestWithId !== null) {
            closestWithId = input[idField] ? { __typename: format.gqlRelMap.__typename, id: input[idField], path: "" } : closestWithId;
        }

        for (const field in input) {
            const action = getActionFromFieldName(field);
            // if not one of the action suffixes, add to the inputInfo object and continue
            if (!action) {
                inputInfo.input[field] = input[field];
                continue;
            }

            const fieldName = field.substring(0, field.length - action.length);
            let __typename: `${GqlModelType}` = format.gqlRelMap[fieldName] as `${GqlModelType}`;
            // Make sure that the relation is defined in the relMap
            if (typeof __typename !== "string") {
                // Translations are a special case, as they are handled internally by the object's shaping functions
                if (fieldName === "translations") {
                    // Add to input and continue
                    inputInfo.input[field] = input[field];
                    continue;
                }
                // It might not be an error yet, as the field might be a union
                let handled = false;
                if (format.unionFields) {
                    // Loop through union fields
                    for (const [unionField, unionFieldValue] of Object.entries(format.unionFields)) {
                        if (!unionFieldValue) continue;
                        // The union field should always exist in gqlRelMap. If not, the format is configured incorrectly.
                        const unionMap = format.gqlRelMap[unionField];
                        if (!unionMap) {
                            throw new CustomError("0527", "InternalError", ["en"], { field });
                        }
                        // There are two possible formats for the unionField data:
                        // 1. An empty object - This means that everything we need is in the unionMap
                        // 2. An object with `connectField` and `relField` properties - This means that the input uses two fields to represent every union connection. 
                        // For example, a Bookmark could have the input fields `forConnect` and `bookmarkFor`, instead of listing a relation connection field for 
                        // every relation type (which would be a lot of fields).
                        const isSimpleUnion = Object.keys(unionFieldValue).length === 0;
                        // Handle simple union
                        if (isSimpleUnion) {
                            // Check if the unionMap contains the field we're looking for
                            if (unionMap[fieldName]) {
                                // If so, we found the union type
                                __typename = unionMap[fieldName] as `${GqlModelType}`;
                                handled = true;
                            }
                        }
                        // Handle complex union
                        else {
                            // Check if the connectField is the field we're looking for
                            if (unionFieldValue.connectField === (field as string)) {
                                // Make sure that typeField is also in the input. If not, something is wrong with the input.
                                if (!input[unionFieldValue.typeField as string]) {
                                    throw new CustomError("0528", "InternalError", ["en"], { field });
                                }
                                // If so, we found the union type
                                __typename = input[unionFieldValue.typeField as string] as `${GqlModelType}`;
                                handled = true;
                            }
                        }
                        if (handled) break;
                    }
                }
                // If we didn't handle the missing __typename, throw an error
                if (!handled) {
                    throw new CustomError("0525", "InternalError", ["en"], { field });
                }
            }
            const { format: childFormat, idField: childIdField } = ModelMap.getLogic(["format", "idField"], __typename, true, "cudInputsToMaps buildTree");

            const processObject = (childInput: object) => {
                // Recursively build child nodes for Create and Update actions
                if (action === "Create" || action === "Update") {
                    const childNode = buildTree(
                        action,
                        childInput,
                        childFormat,
                        languages,
                        childIdField,
                        closestWithId === null ? null : { ...closestWithId, path: closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName },
                    );
                    childNode.parent = rootNode;
                    rootNode.children.push(childNode);
                }
            };

            const processConnectOrDisconnect = (id: string, isToOne: boolean) => {
                initByAction(action);
                initByType(__typename);
                // Connect should still be an ID, so we can add it to the idsByAction and idsByType maps
                // Disconnect should be "true", so we can ignore it
                if (action === "Connect") {
                    idsByAction[action]?.push(id);
                    idsByType[__typename]?.push(id);
                }
                // If closestWithId is null, this means this action takes place within a create mutation. 
                // When this is the case, there are no implicit connects/disconnects, so we can skip adding placeholders.
                if (closestWithId === null) {
                    return;
                }
                // If here, we're connecting or disconnecting a relation within an update mutation. 
                // If this is one-to-one or many-to-one, we need to generate a placeholder.
                // This is because we may be kicking-out an existing object without knowing its ID, and we need to query the database to find it.
                if (isToOne) {
                    const placeholder = `${closestWithId.__typename}|${closestWithId.id}.${closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName}`;
                    idsByAction[action]?.push(placeholder);
                    idsByType[__typename]?.push(placeholder);
                }
            };

            const isArray = (input[field] as unknown) instanceof Array;
            const isConnectOrDisconnect = action === "Connect" || action === "Disconnect";
            if (isArray && !isConnectOrDisconnect) {
                inputInfo.input[field] = [];
                for (const child of input[field] as Array<object>) {
                    processObject(child);
                    inputInfo.input[field].push(child[childIdField]);
                }
            } else if (!isArray && !isConnectOrDisconnect) {
                processObject(input[field] as object);
                inputInfo.input[field] = (input[field] as object)[childIdField];
            } else if (isArray && isConnectOrDisconnect) {
                inputInfo.input[field] = input[field];
                for (const child of input[field] as Array<string>) {
                    processConnectOrDisconnect(child, false);
                }
            } else {
                inputInfo.input[field] = input[field];
                processConnectOrDisconnect(input[field] as string, true);
            }

        }
        // Add inputInfo to inputsById and inputsByType
        // Check if inputInfo already exists in inputsById
        if (input[idField] in inputsById) {
            console.warn("TODO Not sure if this is a problem");
        }
        inputsById[input[idField]] = inputInfo;
        inputsByType[format.gqlRelMap.__typename]?.[actionType]?.push(inputInfo);
        // Return the root node
        return rootNode;
    };

    for (const { actionType, input, objectType } of inputData) {
        // If input is a string (i.e. an ID instead of an object), add it to the idsByAction and idsByType maps without further processing
        // TODO this is fine for "Delete", but "Connect" and "Disconnect" should also do placeholder stuff
        if (typeof input === "string") {
            idsByType[objectType] = (idsByType[objectType] || []).concat(input);
            idsByAction[actionType] = (idsByAction[actionType] || []).concat(input);
        }
        // Otherwise, recursively build the tree and other maps 
        else {
            const { format, idField } = ModelMap.getLogic(["format", "idField"], objectType, true, "cudInputsToMaps 2");
            buildTree(actionType as QueryAction, input, format, languages, idField);
        }
    }

    await convertPlaceholders({ idsByAction, idsByType, prisma, languages });

    // Check if any objects being created should be replaced with connects
    // NOTE: This only updates the maps, not inputs themselves. You'll still have to 
    // perform the correct checks when shaping the inputs.
    for (const type in inputsByType) {
        // Check if the type can be converted to a connect
        const { mutate } = ModelMap.getLogic(["mutate"], type as GqlModelType, true, "cudInputsToMaps connect check");
        if (!mutate?.shape?.findConnects) continue;
        // Collect all IDs of this type being created
        const createIds = inputsByType[type].Create.map(({ node }) => node.id);
        if (!createIds?.length) continue;
        // Find all objects of this type that already exist
        const connectIds = await mutate.shape.findConnects({ Create: inputsByType[type].Create, prisma });
        for (let i = 0; i < createIds.length; i++) {
            const createId = createIds[i];
            const connectId = connectIds[i];
            // If null, an existing ID wasn't found. So skip.
            if (!connectId) continue;
            // Add to idsCreateToConnect
            idsCreateToConnect[createId] = connectId;
            // Move node and change it to a connect
            const inputInfo = inputsById[connectId];
            inputsByType[type].Create.splice(i, 1);
            inputInfo.node.action = "Connect";
            inputInfo.input = connectId;
            inputsByType[type].Connect.push(inputInfo);
            // Remove from idsByAction.Create and add to idsByAction.Connect
            const idIndex = idsByAction.Create?.indexOf(createId);
            if (idIndex === undefined || idIndex === -1 || !idsByAction.Create) continue;
            idsByAction.Create.splice(idIndex, 1);
            idsByAction.Connect = (idsByAction.Connect || []).concat(connectId);
        }
    }

    // Remove duplicate IDs from idsByAction and idsByType
    for (const type in idsByType) {
        idsByType[type] = [...new Set(idsByType[type])];
    }
    for (const action in idsByAction) {
        idsByAction[action] = [...new Set(idsByAction[action])];
    }

    return {
        idsByType,
        idsByAction,
        idsCreateToConnect,
        inputsById,
        inputsByType,
    };
};
