import { GqlModelType } from "@local/shared";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { GqlRelMap } from "../models/types";
import { PrismaType } from "../types";
import { convertPlaceholders } from "./convertPlaceholders";
import { IdsByAction, IdsByType, QueryAction } from "./types";

class InputNode {
    __typename: string;
    id: string;
    action: QueryAction;
    data: any;
    children: InputNode[];
    parent: InputNode | null;

    constructor(__typename: string, id: string, action: QueryAction, data: any) {
        this.__typename = __typename;
        this.id = id;
        this.action = action;
        this.data = data;
        this.children = [];
        this.parent = null;
    }

    getId(): string {
        return this.id;
    }

    getAction(): QueryAction {
        return this.action;
    }

    addChild(childNode: InputNode): void {
        this.children.push(childNode);
        childNode.parent = this;
    }

    setParent(parentNode: InputNode): void {
        this.parent = parentNode;
        parentNode.children.push(this);
    }

    removeChild(childNode: InputNode): void {
        const index = this.children.indexOf(childNode);
        if (index !== -1) {
            this.children.splice(index, 1);
            childNode.parent = null;
        }
    }

    findChild(__typename: string, id: string): InputNode | null {
        return this.children.find(child => child.__typename === __typename && child.id === id) || null;
    }

    hasChildren(): boolean {
        return this.children.length > 0;
    }
}
export type RootNodesById = { [key: string]: InputNode };

// Helper function to derive the action from the field name
function getActionFromFieldName(fieldName) {
    const actions = ["Connect", "Create", "Delete", "Disconnect", "Update"];
    for (const action of actions) {
        if (fieldName.endsWith(action)) {
            return action;
        }
    }
    return null;
}

// TODO for morning:
// 1. Check if this should be handling the updates shaped like ({ where: any, data: any}). The original does this.
// 1.5. Need to handle union types (i.e. gqlRelMap value is an object instead of a string)
// 2. Try adding `readMany` so we can use this for reads. I believe the current way we do reads doesn't properly check relation permissions, so this would solve that.
// 3. Test performance of using this approach vs. old by using console.time() and console.timeEnd() around the original function call, and this new one which we can add in cudHelper.
// 4. Test memory usage of using this approach vs. old by using process.memoryUsage() before and after the original function call, and this new one which we can add in cudHelper.
// 5. Improve type safety of this file
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
    idsByType: IdsByType,
    rootNodesById: RootNodesById,
}> => {
    const idsByAction: IdsByAction = {};
    const idsByType: IdsByType = {};
    const rootNodesById = {};

    const inputs = [
        ...((createMany ?? []).map(data => ({ actionType: "Create", data }))),
        ...((updateMany ?? []).map(({ data }) => ({ actionType: "Update", data }))),
        ...((deleteMany ?? []).map(id => ({ actionType: "Delete", data: id }))),
    ];

    /**
     * Recursively builds a hierarchical representation (tree structure) from a mutation input 
     * to optimize the performance of various operations performed before and after creating, 
     * updating, deleting, connecting, and disconnecting objects.
     *
     * Moreover, the function simultaneously builds two maps, `idsByAction` and `idsByType`,
     * categorizing the IDs by their mutation action (like "Create", "Update", etc.) and by their
     * object type respectively. This aids in bulk database queries for validation, triggers, and other 
     * operations.
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
        const rootNode = new InputNode(relMap.__typename, input[idField], actionType, input);

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
            if (!action) continue;  // if not one of the action suffixes, skip

            const fieldName = field.substring(0, field.length - action.length);
            const __typename = relMap[fieldName];
            // Make sure that the relation is defined in the relMap
            if (!__typename) {
                if (field.startsWith("translations")) continue; // Translations are a special case, as they are handled internally by the object's shaping functions
                throw new CustomError("0525", "InternalError", ["en"], { field });
            }
            const { format: childFormat, idField: childIdField } = getLogic(["format", "idField"], __typename, languages, "buildHierarchicalMap loop");

            // Handle array relations
            if (input[field] instanceof Array) {
                idsByAction[action] = (idsByAction[action] || []).concat((input[field] as Array<string | object>).map(item => typeof item === "string" ? item : item[childIdField]));
                idsByType[__typename] = (idsByType[__typename] || []).concat((input[field] as Array<string | object>).map(item => typeof item === "string" ? item : item[childIdField]));

                // Recursively build child nodes for Create and Update actions
                if (action === "Create" || action === "Update") {
                    for (const related of (input[field] as any)) {
                        const childNode = buildTree(
                            action,
                            related,
                            childFormat.gqlRelMap,
                            languages,
                            childIdField,
                            closestWithId === null ? null : { ...closestWithId, path: closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName },
                        );
                        childNode.parent = rootNode;
                        rootNode.children.push(childNode);
                    }
                }
            }
            // Handle non-array relations for Connect and Disconnect
            else if (action === "Connect" || action === "Disconnect") {
                // Connect should still be an ID, so we can add it to the idsByAction and idsByType maps
                // Disconnect should be "true", so we can ignore it
                if (action === "Connect") {
                    idsByAction[action] = (idsByAction[action] || []).concat(input[field] as string);
                    idsByType[__typename] = (idsByType[__typename] || []).concat(input[field] as string);
                }
                // If closestWithId is null, this means this action takes place within a create mutation. 
                // When this is the case, there are no implicit connects/disconnects, so we can skip adding placeholders.
                if (closestWithId === null) {
                    continue;
                }
                // If here, we're connecting or disconnecting a one-to-one or many-to-one relation within an update mutation. 
                // This requires a placeholder, as we may be kicking-out an existing object without knowing its ID, and we need to query the database to find it.
                const placeholder = `${closestWithId.__typename}|${closestWithId.id}.${closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName}`;
                idsByAction[action] = (idsByAction[action] || []).concat(placeholder);
                idsByType[__typename] = (idsByType[__typename] || []).concat(placeholder);
            } else {
                //Shouldn't be here
                console.error("Shouldn't be here");
            }
        }

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
            rootNodesById[inputRootNode.getId()] = inputRootNode;
        }
    }

    const withoutPlaceholders = await convertPlaceholders({
        idsByAction,
        idsByType,
        prisma,
        languages,
    });

    return {
        idsByType: withoutPlaceholders.idsByType,
        idsByAction: withoutPlaceholders.idsByAction,
        rootNodesById,
    };
};
