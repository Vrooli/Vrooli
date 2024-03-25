import { DUMMY_ID, GqlModelType } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { modelToGql } from "../builders/modelToGql";
import { selectHelper } from "../builders/selectHelper";
import { PartialGraphQLInfo, PrismaCreate, PrismaUpdate } from "../builders/types";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { PreMap } from "../models/types";
import { SessionUserToken } from "../types";
import { cudInputsToMaps } from "../utils/cudInputsToMaps";
import { cudOutputsToMaps } from "../utils/cudOutputsToMaps";
import { getAuthenticatedData } from "../utils/getAuthenticatedData";
import { CudInputData } from "../utils/types";
import { maxObjectsCheck } from "../validators/maxObjectsCheck";
import { permissionsCheck } from "../validators/permissions";
import { profanityCheck } from "../validators/profanityCheck";

/**
 * Performs create, update, and delete operations. 
 * 
 * First performs the following validations, which also validates all relationships:  
 * - Yup validation (e.g. required fields, string min/max length)
 * - Detecting profanity in translation fields
 * - Ownership validation (e.g. user is owner of object, or user is a member of the organization with permission to edit/delete) 
 * - Transfer ownership validation
 * - Max count validation (e.g. user can't have more than 100 organizations)
 * - Other custom validation
 * 
 * Then, it shapes create and update data to be inserted into the database. 
 * Sometimes, creates are converted to connects, and updates are converted to disconnects and connects.
 * 
 * Finally, it performs the operations in the database, and returns the results in shape of GraphQL objects. 
 * Results are in the same order as the input data.
 */
export async function cudHelper({
    inputData,
    partialInfo,
    userData,
}: {
    inputData: CudInputData[],
    partialInfo: PartialGraphQLInfo,
    userData: SessionUserToken,
}): Promise<Array<boolean | Record<string, any>>> {
    // Initialize results
    const result: Array<boolean | Record<string, any>> = new Array(inputData.length).fill(false);
    // Validate and cast inputs
    for (let i = 0; i < inputData.length; i++) {
        const { action, input, objectType } = inputData[i];
        if (action === "Create") {
            const { mutate } = ModelMap.getLogic(["mutate"], objectType, true, "cudHelper cast create");
            const transformedInput = mutate.yup.create && mutate.yup.create({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
            inputData[i].input = transformedInput;
        } else if (action === "Update") {
            const { mutate } = ModelMap.getLogic(["mutate"], objectType, true, "cudHelper cast update");
            const transformedInput = mutate.yup.update && mutate.yup.update({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
            inputData[i].input = transformedInput;
        }
    }
    // Group all data, including relations, relations' relations, etc. into various maps. 
    // These are useful for validation and pre-shaping data
    const {
        idsByAction,
        idsByType,
        idsCreateToConnect,
        inputsById,
        inputsByType,
    } = await cudInputsToMaps({ inputData });
    const preMap: PreMap = {};
    // For each type, calculate pre-shape data (if applicable). 
    // This often also doubles as a way to perform custom input validation
    for (const [type, inputs] of Object.entries(inputsByType)) {
        // Initialize type as empty object
        preMap[type] = {};
        const { mutate } = ModelMap.getLogic(["mutate"], type as GqlModelType, true, "cudHelper preshape type");
        if (mutate.shape.pre) {
            const preResult = await mutate.shape.pre({ ...inputs, userData, inputsById });
            preMap[type] = preResult;
        }
    }
    // Query for all authentication data
    const authDataById = await getAuthenticatedData(idsByType, userData);
    // Validate permissions
    await permissionsCheck(authDataById, idsByAction, inputsById, userData);
    // Perform profanity checks
    profanityCheck(inputData, inputsById, authDataById, userData.languages);
    // Max objects check
    await maxObjectsCheck(inputsById, authDataById, idsByAction, userData);
    // Group top-level (i.e. can't use inputsByType, idsByAction, etc. because those include relations) data by type
    const topInputsByType: { [key in GqlModelType]?: {
        Create: { index: number, input: PrismaCreate }[],
        Update: { index: number, input: PrismaUpdate }[],
        Delete: { index: number, input: string }[],
    } } = {};
    for (const [index, { action, input, objectType }] of inputData.entries()) {
        if (!topInputsByType[objectType]) {
            topInputsByType[objectType] = { Create: [], Update: [], Delete: [] };
        }
        topInputsByType[objectType]![action].push({ index, input: input as any });
    }
    // Initialize data for afterMutations trigger
    const deletedIdsByType: { [key in GqlModelType]?: string[] } = {};
    const beforeDeletedData: { [key in GqlModelType]?: object } = {};
    // Create array to store operations for transaction
    const operations: PrismaPromise<any>[] = [];
    // Loop through each type to populate operations array
    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        const { delegate, idField, mutate } = ModelMap.getLogic(["delegate", "idField", "mutate"], objectType as GqlModelType, true, "cudHelper loop");
        const deletingIds = Delete.map(({ input }) => input);
        // Create
        if (Create.length > 0) {
            for (const { index } of Create) {
                const { input, objectType } = inputData[index] as { input: PrismaCreate, objectType: GqlModelType | `${GqlModelType}` };
                // Make sure no objects with placeholder ids are created. These could potentially bypass permissions/api checks, 
                // since they're typically used to satisfy validation for relationships that aren't needed for the create 
                // (e.g. `listConnect` on a resource item that's already being created in a list)
                if (input?.id === DUMMY_ID) {
                    throw new CustomError("0501", "InternalError", userData.languages, { input, objectType });
                }
                const data = mutate.shape.create ? await mutate.shape.create({ data: input, idsCreateToConnect, preMap, userData }) : input;
                const select = selectHelper(partialInfo)?.select;
                // Add to operations
                const createOperation = delegate(prismaInstance).create({
                    data,
                    select,
                });
                operations.push(createOperation as any);
            }
        }
        // Update
        if (Update.length > 0) {
            for (const { index } of Update) {
                const { input } = inputData[index] as { input: PrismaUpdate, objectType: GqlModelType | `${GqlModelType}` };
                const data = mutate.shape.update ? await mutate.shape.update({ data: input, idsCreateToConnect, preMap, userData }) : input;
                const select = selectHelper(partialInfo)?.select;
                // Add to operations
                const updateOperation = delegate(prismaInstance).update({
                    where: { [idField]: input[idField] },
                    data,
                    select,
                });
                operations.push(updateOperation as any);
            }
        }
        // Delete
        if (Delete.length > 0) {
            // Call beforeDeleted
            if (mutate.trigger?.beforeDeleted) {
                await mutate.trigger.beforeDeleted({ beforeDeletedData, deletingIds, userData });
            }
            // Delete
            try {
                // Before deleting, check which ids exist
                const existingIds: string[] = await delegate(prismaInstance).findMany({
                    where: { [idField]: { in: deletingIds } },
                    select: { id: true },
                }).then(x => x.map(({ id }) => id));
                // Update deletedIdsByType
                deletedIdsByType[objectType] = existingIds;
                // Add to operations
                const deleteOperation = delegate(prismaInstance).deleteMany({
                    where: { [idField]: { in: existingIds } },
                });
                operations.push(deleteOperation as any);
            } catch (error) {
                throw new CustomError("0417", "InternalError", userData.languages, { error, deletingIds, objectType });
            }
        }
    }
    // Perform all operations in transaction
    const transactionResult = await prismaInstance.$transaction(operations);
    // Loop again through each type to process results
    let transactionIndex = 0;
    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        for (let i = 0; i < Create.length; i++) {
            const createdObject = transactionResult[transactionIndex];
            const converted = modelToGql<object>(createdObject, partialInfo);
            result[Create[i].index] = converted;
            transactionIndex++;
        }
        for (let i = 0; i < Update.length; i++) {
            const updatedObject = transactionResult[transactionIndex];
            const converted = modelToGql<object>(updatedObject, partialInfo);
            result[Update[i].index] = converted;
            transactionIndex++;
        }
        if (Delete.length > 0) {
            const ids = deletedIdsByType[objectType];
            for (const id of ids) {
                const index = inputData.findIndex(({ input }) => input === id);
                if (index >= 0) result[index] = true;
            }
            transactionIndex++;  // No need to iterate over multiple deletes since deleteMany returns a count, not individual records
        }
    }
    // Similar to how we grouped inputs, now we need to group outputs
    const outputsByType = cudOutputsToMaps({ idsByAction, inputsById });
    // Call afterMutations function for each type
    const allTypes = new Set([...Object.keys(outputsByType), ...Object.keys(deletedIdsByType)]);
    for (const type of allTypes) {
        const { mutate } = ModelMap.getLogic(["mutate"], type as GqlModelType, true, "cudHelper afterMutations type");
        if (!mutate.trigger?.afterMutations) continue;
        const createInputs = outputsByType[type]?.createInputs || [];
        const createdIds = outputsByType[type]?.createdIds || [];
        const updatedIds = outputsByType[type]?.updatedIds || [];
        const updateInputs = outputsByType[type]?.updateInputs || [];
        const deletedIds = deletedIdsByType[type as GqlModelType] || [];
        await mutate.trigger.afterMutations({
            beforeDeletedData,
            createdIds,
            createInputs,
            deletedIds,
            preMap,
            updatedIds,
            updateInputs,
            userData,
        });
    }
    return result;
}
