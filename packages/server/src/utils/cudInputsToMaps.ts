import { ModelType, pascalCase, validatePK } from "@local/shared";
import { isRelationshipObject } from "../builders/isOfType.js";
import { PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";
import { Formatter, ModelLogicType, PreMap } from "../models/types.js";
import { getActionFromFieldName } from "./getActionFromFieldName.js";
import { InputNode } from "./inputNode.js";
import { CudInputData, IdsByAction, IdsByPlaceholder, IdsByType, IdsCreateToConnect, InputsById, InputsByType, QueryAction, ResultsById } from "./types.js";

/** Information about the closest known object with a valid, **existing** ID (i.e. not a new object) in a mutation */
type ClosestWithId = { __typename: string, id: string, path: string };

/** Formatter with only the data we need to generate input maps */
type MinimumFormatter<
    Model extends {
        __typename: `${ModelType}`,
        ApiCreate: ModelLogicType["ApiCreate"],
        ApiModel: ModelLogicType["ApiModel"],
        DbModel: ModelLogicType["DbModel"],
    }
> = Pick<Formatter<Model>, "apiRelMap" | "unionFields">;

/**
 * Fetches and maps a placeholder to its corresponding unique identifier.
 * Updates `placeholderToIdMap` with the results.
 *
 * This function takes a placeholder string, which encodes information about an object and its relations,
 * and resolves it to a unique identifier (ID) by querying the database using the Prisma client. If the
 * placeholder has already been processed and mapped, it retrieves the ID from the existing map to avoid
 * redundant database queries. 
 *
 * @param placeholder - A string representing the placeholder to be resolved. The format is
 *                      "objectType|rootId.relationType1|relation1.relationType2|relation2...relationTypeN|relationN".
 * @param placeholderToIdMap - A map object (IdsByPlaceholder) that stores previously resolved placeholders
 *                             and their corresponding IDs to optimize performance.
 * @returns {Promise<string | null>} - The resolved unique identifier (ID) of the object represented by
 *                                     the placeholder, or null if the object is not found.
 *
 * @example
 * // Assuming 'User|123.Email|emails' is a placeholder where 'User' is the object type, '123' is the rootId,
 * // 'Email' is the relation type for 'email'.
 * const userId = await fetchAndMapPlaceholder('User|123.Email|emails', placeholderToIdMap);
 */
export async function fetchAndMapPlaceholder(
    placeholder: string,
    placeholderToIdMap: IdsByPlaceholder,
): Promise<void> {
    if (placeholder in placeholderToIdMap) {
        return;  // Already processed this placeholder
    }

    const parts = placeholder.split(".");
    const [objectType, rootId] = parts[0].split("|", 2);

    // Check if the placeholder is just an ID, in which case
    // we'll add it to the map and return
    if (parts.length === 1 && validatePK(rootId)) {
        logger.warning("Unnecessary placeholder was generated. This may be a bug", { trace: "0163", placeholder });
        // Directly map the rootId as the ID without querying the database
        placeholderToIdMap[placeholder] = rootId;
        return;
    }

    const { dbTable, format: _format } = ModelMap.getLogic(["dbTable", "format"], pascalCase(objectType) as ModelType, true, "fetchAndMapPlaceholder 1");

    // Construct the select object to query nested relations
    const select: Record<string, any> = {};
    let currentSelect = select;
    parts.forEach((part, index) => {
        if (index === 0) {
            currentSelect.id = true; // Add root ID selection
        } else {
            const [relationType, relation] = part.split("|");
            const { idField } = ModelMap.getLogic(["idField"], relationType as ModelType, false, "fetchAndMapPlaceholder 2");
            currentSelect[relation] = { select: { [idField ?? "id"]: true } };
            currentSelect = currentSelect[relation].select;
        }
    });

    const queryResult = await (DbProvider.get()[dbTable] as PrismaDelegate).findUnique({
        where: { id: rootId },
        select,
    });

    let currentObject = queryResult;
    for (const part of parts.slice(1)) {
        const [, relation] = part.split("|");
        if (!currentObject) {
            break; // Break the loop if currentObject is null or undefined
        }
        currentObject = currentObject[relation];
    }

    const resultId = currentObject && currentObject.id ? currentObject.id : null;
    if (resultId) {
        placeholderToIdMap[placeholder] = resultId;
    }
}

/**
 * Iterates over a given map of IDs, replacing any placeholder IDs with actual IDs fetched from the database.
 * This function is primarily used for converting placeholder IDs in the `idsByType` and `idsByAction` maps
 * into their corresponding actual IDs. It mutates the input `idsMap` by updating the placeholder IDs in-place.
 */
export async function replacePlaceholdersInMap(
    idsMap: Record<string, (string | null)[]>,
    placeholderToIdMap: IdsByPlaceholder,
): Promise<void> {
    for (const [key, ids] of Object.entries(idsMap)) {
        const updatedIds: (string | null)[] = [];
        for (const id of ids) {
            if (typeof id === "string" && id.includes("|")) {
                if (placeholderToIdMap[id]) {
                    updatedIds.push(placeholderToIdMap[id]);
                } else {
                    await fetchAndMapPlaceholder(id, placeholderToIdMap);
                    updatedIds.push(placeholderToIdMap[id]);
                }
            } else {
                updatedIds.push(id);
            }
        }
        idsMap[key] = updatedIds;
    }
}

/**
 * Replaces placeholder IDs with actual IDs in the `inputsById` map.
 * Placeholder IDs are expected to have a specific format (e.g., "user|123.prof|profile")
 * and this function will replace them with actual IDs fetched through some mechanism,
 * updating both the keys in the `inputsById` map and the `id` property of the `node` within each value.
 */
export async function replacePlaceholdersInInputsById(
    inputsById: InputsById,
    placeholderToIdMap: IdsByPlaceholder,
): Promise<void> {
    for (const [maybePlaceholder, value] of Object.entries(inputsById)) {
        if (!maybePlaceholder.includes("|")) continue;
        let id: string | null = maybePlaceholder;
        if (placeholderToIdMap[maybePlaceholder]) {
            id = placeholderToIdMap[maybePlaceholder];
        } else {
            await fetchAndMapPlaceholder(maybePlaceholder, placeholderToIdMap);
            id = placeholderToIdMap[maybePlaceholder];
        }
        if (id === null || id === undefined) {
            logger.warning("Placeholder ID could not be resolved. This may be a bug, or the object may not exist", { trace: "0167", maybePlaceholder });
            delete inputsById[maybePlaceholder];
            continue;
        }

        inputsById[id] = {
            node: {
                ...value.node,
                id,
            },
            input: typeof value.input === "string" ?
                id : isRelationshipObject(value.input) ? {
                    ...value.input,
                    id,
                } : value.input,
        };
        delete inputsById[maybePlaceholder];
    }
}

/**
 * Replaces placeholder IDs with actual IDs in the `inputsByType` structure.
 * This function iterates over each type, action, and their corresponding inputs,
 * updating placeholder IDs in both node.id and input fields.
 */
export async function replacePlaceholdersInInputsByType(
    inputsByType: InputsByType,
    placeholderToIdMap: IdsByPlaceholder,
): Promise<void> {
    for (const objectType in inputsByType) {
        for (const action in inputsByType[objectType]) {
            const inputs = inputsByType[objectType][action];
            for (const inputWrapper of inputs) {
                const maybePlaceholder = inputWrapper.node.id;

                // Add detailed logging to debug undefined id issue
                if (maybePlaceholder === undefined) {
                    console.error("[replacePlaceholdersInInputsByType] ERROR: inputWrapper.node.id is undefined!", {
                        objectType,
                        action,
                        inputWrapper,
                        nodeDetails: inputWrapper.node,
                    });
                    // Continue with the next iteration to prevent further errors
                    continue;
                }

                if (!maybePlaceholder.includes("|")) continue;

                let id = placeholderToIdMap[maybePlaceholder];
                if (!id) {
                    await fetchAndMapPlaceholder(maybePlaceholder, placeholderToIdMap);
                    id = placeholderToIdMap[maybePlaceholder];
                }
                if (id === null || id === undefined) {
                    logger.warning("Placeholder ID could not be resolved. This may be a bug, or the object may not exist", { trace: "0169", maybePlaceholder });
                }

                inputWrapper.node.id = id;
                inputWrapper.input = typeof inputWrapper.input === "string" ?
                    id : isRelationshipObject(inputWrapper.input) ? {
                        ...inputWrapper.input,
                        id,
                    } : inputWrapper.input;
            }
        }
    }
}

/**
 * Converts placeholder ids to actual IDs, or null if actual ID not found. 
 * This is only needed for idsByAction and idsByType
 */
export async function convertPlaceholders({
    idsByAction,
    idsByType,
    inputsById,
    inputsByType,
}: {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsById: InputsById,
    inputsByType: InputsByType,
}): Promise<void> {
    const placeholderToIdMap: IdsByPlaceholder = {};

    await replacePlaceholdersInMap(idsByAction, placeholderToIdMap);
    await replacePlaceholdersInMap(idsByType, placeholderToIdMap);
    await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);
    await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);
}

/**
 * Initializes and populates maps for tracking IDs and inputs by action type and object type.
 * 
 * @param action - The type of action, such as "Create", "Update", etc.
 * @param objectType - The GraphQL model type, e.g., "User", "Post".
 * @param idsByAction - A map of action types to arrays of IDs.
 * @param idsByType - A map of object types to arrays of IDs.
 * @param inputsByType - A map of object types to arrays of mutation inputs.
 */
export function initializeInputMaps(
    action: QueryAction,
    objectType: `${ModelType}`,
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsByType: InputsByType,
): void {
    // Initialize idsByAction for the given action if it doesn't exist
    if (!idsByAction[action]) {
        idsByAction[action] = [];
    }

    // Initialize idsByType and inputsByType for the given objectType if they don't exist
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
}

/**
 * Updates the closest known object ID for generating input map placeholders.
 * This function is used to track the nearest object with a known ID in the input hierarchy,
 * which is crucial for generating placeholders with enough information to query the database.
 * 
 * @param action - The type of mutation action, such as "Create", "Update", etc.
 * @param input - The GraphQL mutation input object, which can be a string or an object with an ID field.
 * @param idField - The name of the ID field for the current object type.
 * @param inputType - The GraphQL model type, e.g., "User", "Post".
 * @param closestWithId - The current closest object with an ID, used for generating placeholders.
 * @param relation - The relation name, if not the root object (so we can update the path if no ID is found at this level).
 * @returns The updated closestWithId object or null.
 */
export function updateClosestWithId<T extends { [key: string]: any }>(
    action: QueryAction,
    input: string | boolean | T,
    idField: string,
    inputType: ModelType | `${ModelType}`,
    closestWithId: ClosestWithId | null,
    relation?: string,
): ClosestWithId | null {
    // Placeholders are only used for implicit connects/disconnects on one-to-one and many-to-one relations. 
    // Since create mutations don't have any implicit connects/disconnects (since that implies existing relations),
    // we can safely ignore the ID.
    if (action === "Create") {
        return null;
    }
    const inputId: string | null | undefined = typeof input === "string" ?
        input :
        typeof input === "boolean" ?
            null :
            input[idField];
    // If there is no ID
    if (typeof inputId !== "string") {
        // If there is a relation, return with updated path
        if (relation) {
            return closestWithId ?
                { ...closestWithId, path: closestWithId.path.length ? `${closestWithId.path}.${relation}` : relation } :
                null;
        }
        // Otherwise, return null
        return null;
    }
    // If there is an ID, return with ID, inputType, and empty path
    return { __typename: inputType, id: inputId, path: "" };
}

/**
 * Determines the GraphQL model type (`__typename`) for a given field within an input object.
 * This function handles straightforward field-to-type mappings, special cases like translations, 
 * and more complex scenarios involving union fields.
 * 
 * @returns An object containing the determined type, or null if we should treat the field as a non-relation 
 * and skip further processing.
 * @throws {CustomError} If the field is not found in the relMap, or if the field is a union and the union type cannot be determined.
 */
export function determineModelType<
    ApiModel extends ModelLogicType["ApiModel"],
>(
    field: string,
    fieldName: string,
    input: ApiModel,
    format: MinimumFormatter<any>,
): `${ModelType}` | null {
    // Check if the field exists in the relMap (the standard case)
    const __typename: `${ModelType}` = format.apiRelMap[fieldName] as `${ModelType}`;
    if (typeof __typename === "string") {
        return __typename;
    }
    // Check for special cases
    // Special case 1: Translations. These are handled internally by the object's shaping functions, 
    // which means we can add it to inputInfo skip the rest of the processing.
    if (fieldName === "translations") {
        return null;
    }
    // If we get here and there's no union data, something is wrong with either the input 
    // or the format. Throw an error.
    if (!format.unionFields) {
        throw new CustomError("0525", "InternalError", { field, fieldName });
    }
    // Loop through union fields
    for (const [unionField, unionFieldValue] of Object.entries(format.unionFields)) {
        if (!unionFieldValue) continue;
        // The union field should always exist in apiRelMap. If not, the format is configured incorrectly.
        const unionMap = format.apiRelMap[unionField];
        if (!unionMap) {
            throw new CustomError("0527", "InternalError", { field, fieldName });
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
                return unionMap[fieldName] as `${ModelType}`;
            }
            continue;
        }
        // Handle complex union
        // Check if the connectField is the field we're looking for
        if (unionFieldValue.connectField === field) {
            // Make sure that typeField is also in the input. If not, something is wrong with the input.
            if (!input[unionFieldValue.typeField as string]) {
                throw new CustomError("0488", "InternalError", { field, fieldName });
            }
            // If so, we found the union type
            return input[unionFieldValue.typeField as string] as `${ModelType}`;
        }
    }
    // If we get here, we couldn't find the union type. Throw an error.
    throw new CustomError("0228", "InternalError", { field, fieldName });
}


/**
 * Recursively builds child nodes for "Create" and "Update" actions within the input tree.
 * @returns The newly created child node, which has been integrated into the input tree.
 */
export function processCreateOrUpdate<
    Typename extends `${ModelType}`,
    ApiCreate extends ModelLogicType["ApiCreate"],
    ApiModel extends ModelLogicType["ApiModel"],
    DbModel extends ModelLogicType["DbModel"],
>(
    action: QueryAction,
    input: ApiModel,
    format: MinimumFormatter<{ __typename: Typename, ApiCreate: ApiCreate, ApiModel: ApiModel, DbModel: DbModel }>,
    fieldName: string,
    idField: string,
    parentNode: InputNode,
    closestWithId: { __typename: string, id: string, path: string } | null,
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsById: InputsById,
    inputsByType: InputsByType,
): void {
    // Recursively build child nodes for Create and Update actions
    if (action === "Create" || action === "Update") {
        // Disallow Update if we're in a create mutation
        if (action === "Update" && (closestWithId === null || closestWithId === undefined)) {
            throw new CustomError("0004", "InternalError", { fieldName });
        }
        const childNode = inputToMaps(
            action,
            input,
            format,
            idField,
            (closestWithId === null || closestWithId === undefined) ? null : { ...closestWithId, path: closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName },
            idsByAction,
            idsByType,
            inputsById,
            inputsByType,
        );
        childNode.parent = parentNode;
        parentNode.children.push(childNode);
    } else {
        // If other functions are set up correctly, this should never happen
        throw new CustomError("0110", "InternalError", { action });
    }
}

/**
 * Processes "Connect", "Disconnect", and "Delete" actions by managing IDs in tracking maps
 * and handling placeholders for implicit "Disconnects".
 */
export function processConnectDisconnectOrDelete(
    id: string,
    isToOne: boolean,
    action: QueryAction,
    fieldName: string | null,
    __typename: ModelType | `${ModelType}`,
    parentNode: InputNode,
    closestWithId: { __typename: string, id: string, path: string } | null,
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsById: InputsById,
    inputsByType: InputsByType,
): void {
    initializeInputMaps(action, __typename, idsByAction, idsByType, inputsByType);
    // Check if closestWithId is null. If so, this means this action takes place within a create mutation, 
    // so we can skip some steps.
    const isInCreate = (closestWithId === null || closestWithId === undefined);
    // Disallow Disconnect and Delete if we're in a create mutation
    if (["Disconnect", "Delete"].includes(action) && isInCreate) {
        throw new CustomError("0111", "InternalError", { trace: "0124", id });
    }
    // Handle placeholders first.
    // Placeholders are only used for implicit disconnects/deletes on one-to-one and many-to-one relations.
    // This is because we may be kicking-out an existing object without knowing its ID, and we need to query the database to find it.
    if (!isInCreate && isToOne && fieldName) {
        const newSegment = `${__typename}|${fieldName}`;
        const placeholder = `${closestWithId.__typename}|${closestWithId.id}.${closestWithId.path.length ? `${closestWithId.path}.${newSegment}` : newSegment}`;
        // const placeholder = `${closestWithId.__typename}|${closestWithId.id}.${closestWithId.path.length ? `${closestWithId.path}.${fieldName}` : fieldName}`;
        let placeholderAction: "Disconnect" | "Delete" | null = null;
        // Connect and Disconnect may be implicitly DISCONNECTING the previous relation
        if (["Connect", "Disconnect"].includes(action)) {
            placeholderAction = "Disconnect";
        }
        // Delete implicitly DELETES the previous relation. This is because for -to-one 
        // relations, we're providing a boolean instead of an ID, so we don't know the ID.
        else if (action === "Delete") {
            placeholderAction = "Delete";
        }
        if (placeholderAction) {
            initializeInputMaps(placeholderAction, __typename, idsByAction, idsByType, inputsByType);
            idsByAction[placeholderAction]?.push(placeholder);
            idsByType[__typename]?.push(placeholder);
            const node = new InputNode(__typename, placeholder, placeholderAction);
            node.parent = parentNode;
            parentNode.children.push(node);
            // We only need to add to inputsById if it's a "Delete" action, 
            // since only "Create", "Update", and "Delete" operations use the input tree.
            if (placeholderAction === "Delete") {
                const inputInfo = { node, input: placeholder };
                inputsById[placeholder] = inputInfo;
                inputsByType[__typename]?.[placeholderAction]?.push(inputInfo);
            }
        }
    }
    // Now handle non-placeholder cases by adding ID and node to maps. 
    // We'll skip this for isToOne Disconnects and Deletes, since they don't have an ID 
    // (they use a boolean instead)
    if (!(isToOne && ["Disconnect", "Delete"].includes(action))) {
        idsByAction[action]?.push(id);
        idsByType[__typename]?.push(id);
        const node = new InputNode(__typename, id, action);
        node.parent = parentNode;
        parentNode.children.push(node);
        // We only need to add to inputsById if it's a "Delete" action, 
        // since only "Create", "Update", and "Delete" operations use the input tree.
        if (action === "Delete") {
            const inputInfo = { node, input: id };
            inputsById[id] = inputInfo;
            inputsByType[__typename]?.[action]?.push(inputInfo);
        }
    }
}

/**
 * Processes a field in an input object, handling action determination,
 * union fields, and initiating the processing of child objects or connections.
 */
export function processInputObjectField<
    Typename extends `${ModelType}`,
    ApiCreate extends ModelLogicType["ApiCreate"],
    ApiModel extends ModelLogicType["ApiModel"],
    DbModel extends ModelLogicType["DbModel"],
>(
    field: string,
    input: ApiModel,
    format: MinimumFormatter<{ __typename: Typename, ApiCreate: ApiCreate, ApiModel: ApiModel, DbModel: DbModel }>,
    parentNode: InputNode,
    closestWithId: { __typename: string, id: string, path: string } | null,
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsById: InputsById,
    inputsByType: InputsByType,
    inputInfo: { node: InputNode, input: string | Record<string, any> },
): void {
    const action = getActionFromFieldName(field);
    const isToOne = !(input[field] instanceof Array);
    // If it's not a relation, add it to the inputInfo object and return
    if (!action) {
        inputInfo.input[field] = input[field];
        return;
    }

    const fieldName = field.substring(0, field.length - action.length);
    const __typename = determineModelType(field, fieldName, input, format);
    // If __typename wasn't found, treat it as a non-relation and skip further processing
    if (!__typename) {
        inputInfo.input[field] = input[field];
        return;
    }

    const { format: childFormat, idField: childIdField } = ModelMap.getLogic(["format", "idField"], __typename, true, "cudInputsToMaps processInputObjectField");
    const isCreateOrUpdate = ["Create", "Update"].includes(action);
    // Handle Create/Update
    if (isCreateOrUpdate) {
        if (!isToOne) {
            inputInfo.input[field] = [];
            for (const childInput of input[field] as Array<object>) {
                processCreateOrUpdate(action, childInput, childFormat, fieldName, childIdField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
                inputInfo.input[field].push(childInput[childIdField]);
            }
        } else {
            inputInfo.input[field] = (input[field] as object)[childIdField];
            processCreateOrUpdate(action, input[field] as object, childFormat, fieldName, childIdField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }
    }
    // Handle Connect/Disconnect/Delete
    else {
        if (!isToOne) {
            inputInfo.input[field] = input[field];
            for (const childId of input[field] as Array<string>) {
                processConnectDisconnectOrDelete(childId, isToOne, action, fieldName, __typename, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            }
        } else {
            inputInfo.input[field] = input[field];
            // A -to-one Disconnect is a boolean, so we'll pass in an empty string instead
            processConnectDisconnectOrDelete(typeof input[field] === "string" ? input[field] : "", isToOne, action, fieldName, __typename, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }
    }
    // TODO not handling Read
}

/**
 * Populates various maps, using a mutation input, to optimize the performance of operations
 * performed before and after creating, updating, deleting, connecting, and disconnecting objects.
 *
 * NOTE: The operations for "Connect" or "Disconnect" on non-array relations (i.e. one-to-one or many-to-one), 
 * may implicitly connect/disconnect an object without providing an ID. In this case, we generate
 * a placeholder. Another function must convert this later, once we've processed all mutation 
 * inputs and have a complete list of placeholders.
 *
 * @param action - The type of mutation action. Example: "Create" 
 * @param input - The GraphQL mutation input object
 * @param format - ModelLogic property which contains formatting information for the current object type
 * @param languages - The languages to use for error messages
 * @param idField - The name of the ID field for the current object type. Defaults to "id".
 * @param closestWithId - Keeps track of the closest known object ID as we traverse the input. Useful 
 * for generating placeholders.
 * @param idsByAction - A map of action types to IDs. Used to keep track of all IDs in the input.
 * @param idsByType - A map of object types to IDs. Used to keep track of all IDs in the input.
 * @param inputsById - A map of IDs to input objects. Used to keep track of all inputs in the input.
 * @param inputsByType - A map of object types to input objects. Used to keep track of all inputs in the input.
 * @returns rootNode - The root of the hierarchical tree representation of the input.
 */
export function inputToMaps<
    Typename extends `${ModelType}`,
    ApiCreate extends ModelLogicType["ApiCreate"],
    ApiModel extends ModelLogicType["ApiModel"],
    DbModel extends ModelLogicType["DbModel"],
>(
    action: QueryAction,
    input: string | ApiModel,
    format: MinimumFormatter<{ __typename: Typename, ApiCreate: ApiCreate, ApiModel: ApiModel, DbModel: DbModel }>,
    idField = "id",
    closestWithId: { __typename: string, id: string, path: string } | null = { __typename: "", id: "", path: "" },
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsById: InputsById,
    inputsByType: InputsByType,
): InputNode {
    // Initialize data
    const id = typeof input === "string" ? input : input[idField];
    const rootNode = new InputNode(format.apiRelMap.__typename, id, action);
    initializeInputMaps(action, format.apiRelMap.__typename, idsByAction, idsByType, inputsByType);

    // Add the current ID to idsByAction and idsByType
    idsByAction[action]?.push(id);
    idsByType[format.apiRelMap.__typename]?.push(id);

    // Update closestWithId for generating placeholders
    closestWithId = updateClosestWithId(action, input, idField, format.apiRelMap.__typename, closestWithId);

    // Initialize object to store processed input info
    const inputInfo: { node: InputNode, input: any } = { node: rootNode, input: {} };

    // If input is not an object (i.e. an ID)
    if (!isRelationshipObject(input)) {
        // Process as a Delete
        inputInfo.input = input;
        processConnectDisconnectOrDelete(id, true, action, null, format.apiRelMap.__typename, rootNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
    } else {
        // Process each field in the input object
        for (const field in input) {
            processInputObjectField(field, input, format, rootNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }
    }

    // Add inputInfo to inputsById and inputsByType
    // Check if inputInfo already exists in inputsById
    if (id in inputsById) {
        console.warn("TODO Not sure if this is a problem");
    }
    inputsById[id] = inputInfo;
    inputsByType[format.apiRelMap.__typename]?.[action]?.push(inputInfo);
    // Return the root node
    return rootNode;
}

export type CudInputsToMapsParams = {
    inputData: CudInputData[],
}

export type CudInputsToMapsResult = {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    idsCreateToConnect: IdsCreateToConnect,
    inputsById: InputsById,
    inputsByType: InputsByType,
    // Populated later
    preMap: PreMap
    // Populated later
    resultsById: ResultsById,
}

// TODO Try adding `readMany` so we can use this for reads. I believe the current way we do reads doesn't properly check relation permissions, so this would solve that.
/**
 * Groups all input data into various maps for validation and data shaping.
 * @param inputData The array of input data to group.
 * @returns An object containing maps grouped by action, type, etc.
 */
export async function cudInputsToMaps({
    inputData,
}: CudInputsToMapsParams): Promise<CudInputsToMapsResult> {
    const idsByAction: IdsByAction = {};
    const idsByType: IdsByType = {};
    const idsCreateToConnect: IdsCreateToConnect = {};
    const inputsById: InputsById = {};
    const inputsByType: InputsByType = {};

    for (const { action, input, objectType } of inputData) {
        const { format, idField } = ModelMap.getLogic(["format", "idField"], objectType, true, "cudInputsToMaps 2");
        inputToMaps(
            action,
            input,
            format as MinimumFormatter<any>,
            idField,
            null,
            idsByAction,
            idsByType,
            inputsById,
            inputsByType,
        );
    }

    await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

    // Check if any objects being created should be replaced with connects
    // NOTE: This only updates the maps, not inputs themselves. You'll still have to 
    // perform the correct checks when shaping the inputs.
    for (const type in inputsByType) {
        // Check if the type can be converted to a connect
        const { mutate } = ModelMap.getLogic(["mutate"], type as ModelType, true, "cudInputsToMaps connect check");
        if (!mutate?.shape?.findConnects) continue;
        // Collect all IDs of this type being created
        const createIds = inputsByType[type].Create.map(({ node }) => node.id);
        if (!createIds?.length) continue;
        // Find all objects of this type that already exist
        const connectIds = await mutate.shape.findConnects({ Create: inputsByType[type].Create });
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
        preMap: {},
        resultsById: {},
    };
}
