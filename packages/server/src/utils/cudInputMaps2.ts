import { GqlModelType } from "@local/shared";
import { PrismaSelect, PrismaUpdate } from "../builders/types";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { GqlRelMap } from "../models/types";
import { PrismaType } from "../types";
import { IdsByAction, IdsByType, QueryAction } from "./types";

class InputNode {
    __typename: string;
    id: string;
    action: QueryAction;
    children: InputNode[];
    parent: InputNode | null;

    constructor(__typename: string, id: string, action: QueryAction) {
        this.__typename = __typename;
        this.id = id;
        this.action = action;
        this.children = [];
        this.parent = null;
    }
}

type RootNodesById = { [key: string]: InputNode };
type InputsByType = { [key in GqlModelType]?: {
    Connect: { node: InputNode, input: string; }[];
    Create: { node: InputNode, input: PrismaUpdate }[];
    Delete: { node: InputNode, input: string; }[];
    Disconnect: { node: InputNode, input: string; }[];
    Read: { node: InputNode, input: PrismaSelect }[];
    Update: { node: InputNode, input: PrismaUpdate }[];
} };
type IdsByPlaceholder = { [key: string]: string | null };


// Helper function to derive the action from the field name
const getActionFromFieldName = (fieldName: string): QueryAction | null => {
    const actions: QueryAction[] = ["Connect", "Create", "Delete", "Disconnect", "Update"];
    for (const action of actions) {
        if (fieldName.endsWith(action)) {
            return action;
        }
    }
    return null;
};

/**
 * Converts placeholder ids to actual IDs, or null if actual ID not found.
 */
const convertPlaceholders = async ({
    idsByAction,
    prisma,
    languages,
}: {
    idsByAction: IdsByAction,
    prisma: PrismaType,
    languages: string[]
}): Promise<IdsByPlaceholder> => {
    const placeholderToIdMap: IdsByPlaceholder = {};

    // Helper function to fetch and map placeholders
    const fetchAndMapPlaceholder = async (placeholder: string): Promise<void> => {
        if (placeholder in placeholderToIdMap) {
            return;  // Already processed this placeholder
        }

        const [objectType, path] = placeholder.split("|", 2);
        const [rootId, ...relations] = path.split(".");

        const { delegate } = getLogic(["delegate"], objectType as GqlModelType, languages, "convertPlaceholders");
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

        placeholderToIdMap[placeholder] = currentObject && currentObject.id ? currentObject.id : null;
    };

    for (const actionType in idsByAction) {
        const ids = idsByAction[actionType];
        for (const id of ids) {
            if (typeof id === "string" && id.includes("|")) {
                await fetchAndMapPlaceholder(id);
            }
        }
    }

    return placeholderToIdMap;
};

// TODO for morning:
// 1. Check if this should be handling the updates shaped like ({ where: any, data: any}). The original does this.
// 1.5. Need to handle union types (i.e. gqlRelMap value is an object instead of a string)
// 2. Try adding `readMany` so we can use this for reads. I believe the current way we do reads doesn't properly check relation permissions, so this would solve that.
// 3. Try rewriting this to that the createMany, updateMany, and deleteMany don't have to be of the same object type. Would be nice to be able to use this function directly when importing data.
export const cudInputsToMaps2 = async ({
    createMany,
    updateMany,
    deleteMany,
    objectType,
    prisma,
    languages,
}: {
    createMany: { [x: string]: any }[] | null | undefined,
    deleteMany: string[] | null | undefined,
    updateMany: {
        where: { [x: string]: any },
        data: { [x: string]: any },
    }[] | null | undefined,
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
    languages: string[]
}): Promise<{
    idsByAction: IdsByAction,
    idsByPlaceholder: IdsByPlaceholder,
    idsByType: IdsByType,
    inputsByType: InputsByType,
    rootNodesById: RootNodesById,
}> => {
    const idsByAction: IdsByAction = {};
    const idsByType: IdsByType = {};
    const inputsByType: InputsByType = {};
    const rootNodesById = {};

    const inputs = [
        ...((createMany ?? []).map(data => ({ actionType: "Create", data }))),
        ...((updateMany ?? []).map(({ data }) => ({ actionType: "Update", data }))),
        ...((deleteMany ?? []).map(id => ({ actionType: "Delete", data: id }))),
    ];

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
     * @param relMap - A map of relation fields to their object types. Example: { 'chat': 'Chat' }
     * @param languages - The languages to use for error messages
     * @param idField - The name of the ID field for the current object type. Defaults to "id".
     * @param closestWithId - Keeps track of the closest known object ID as we traverse the input. Useful 
     *                             for generating placeholders.
     * @returns rootNode - The root of the hierarchical tree representation of the input.
     */
    const buildTree = <T extends object>(
        actionType: QueryAction,
        input: T,
        relMap: GqlRelMap<T, object>,
        languages: string[],
        idField = "id",
        closestWithId: { __typename: string, id: string, path: string } | null = { __typename: "", id: "", path: "" },
    ): InputNode => {
        // Initialize tree
        const rootNode = new InputNode(relMap.__typename, input[idField], actionType);
        // Initialize maps
        initByAction(actionType);
        initByType(relMap.__typename);
        idsByAction[actionType]?.push(input[idField]);
        idsByType[relMap.__typename]?.push(input[idField]);
        // Initialize object that we'll push to inputsByType later
        const inputInfo: { node: InputNode, input: any } = { node: rootNode, input: {} };

        // Update the closestId when a new ID is found, for use in generating placeholders 
        // The only exception is when a "Create" action is found, as we know that the object doesn't exist yet, 
        // and can thus ignore any following "Connect" or "Disconnect" actions.
        if (actionType === "Create") {
            closestWithId = null; // Set to null to ignore any following "Connect" or "Disconnect" actions
        } else if (closestWithId !== null) {
            closestWithId = input[idField] ? { __typename: relMap.__typename, id: input[idField], path: "" } : closestWithId;
        }

        for (const field in input) {
            const action = getActionFromFieldName(field);
            // if not one of the action suffixes, add to the inputInfo object and continue
            if (!action) {
                inputInfo.input[field] = input[field];
                continue;
            }

            const fieldName = field.substring(0, field.length - action.length);
            const __typename = relMap[fieldName];
            // Make sure that the relation is defined in the relMap
            if (!__typename) {
                if (field.startsWith("translations")) continue; // Translations are a special case, as they are handled internally by the object's shaping functions
                throw new CustomError("0525", "InternalError", ["en"], { field });
            }
            const { format: childFormat, idField: childIdField } = getLogic(["format", "idField"], __typename, languages, "buildHierarchicalMap loop");

            const processObject = (childInput: object) => {
                // Recursively build child nodes for Create and Update actions
                if (action === "Create" || action === "Update") {
                    const childNode = buildTree(
                        action,
                        childInput,
                        childFormat.gqlRelMap,
                        languages,
                        childIdField,
                        closestWithId === null ? null : { ...closestWithId, path: closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName },
                    );
                    childNode.parent = rootNode;
                    rootNode.children.push(childNode);
                }
            };

            const processConnectOrDisconnect = (id: string) => {
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
                // If here, we're connecting or disconnecting a one-to-one or many-to-one relation within an update mutation. 
                // This requires a placeholder, as we may be kicking-out an existing object without knowing its ID, and we need to query the database to find it.
                const placeholder = `${closestWithId.__typename}|${closestWithId.id}.${closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName}`;
                idsByAction[action]?.push(placeholder);
                idsByType[__typename]?.push(placeholder);
            };

            const isArray = input[field] instanceof Array;
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
                    processConnectOrDisconnect(child);
                }
            } else {
                inputInfo.input[field] = input[field];
                processConnectOrDisconnect(input[field] as string);
            }

        }
        // Add inputInfo to inputsByType
        inputsByType[relMap.__typename]?.[actionType]?.push(inputInfo);
        // Return the root node
        return rootNode;
    };

    for (const input of inputs) {
        // If input is a string (i.e. an ID instead of an object), add it to the idsByAction and idsByType maps without further processing
        if (typeof input.data === "string") {
            idsByType[objectType] = (idsByType[objectType] || []).concat(input.data);
            idsByAction[input.actionType] = (idsByAction[input.actionType] || []).concat(input.data);
        } else {
            const { format, idField } = getLogic(["format", "idField"], objectType, languages, "getAuthenticatedIds");
            const inputRootNode = buildTree(input.actionType as QueryAction, input.data, format.gqlRelMap, languages, idField);
            rootNodesById[inputRootNode.id] = inputRootNode;
        }
    }

    const idsByPlaceholder = await convertPlaceholders({ idsByAction, prisma, languages });

    return {
        idsByType,
        idsByPlaceholder,
        idsByAction,
        inputsByType,
        rootNodesById,
    };
};
