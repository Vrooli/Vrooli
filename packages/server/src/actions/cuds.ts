import { DUMMY_ID, ModelType, SEEDED_IDS, SessionUser } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { AnyObjectSchema } from "yup";
import { InfoConverter } from "../builders/infoConverter.js";
import { PartialApiInfo, PrismaCreate, PrismaDelegate, PrismaUpdate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { PreMap } from "../models/types.js";
import { CudInputsToMapsResult, cudInputsToMaps } from "../utils/cudInputsToMaps.js";
import { cudOutputsToMaps } from "../utils/cudOutputsToMaps.js";
import { getAuthenticatedData } from "../utils/getAuthenticatedData.js";
import { CudInputData } from "../utils/types.js";
import { maxObjectsCheck } from "../validators/maxObjectsCheck.js";
import { permissionsCheck } from "../validators/permissions.js";
import { profanityCheck } from "../validators/profanityCheck.js";
import { CudAdditionalData } from "./types.js";

type CudHelperParams = {
    /** Additional data that can be passed to ModelLogic functions */
    additionalData?: CudAdditionalData,
    /**
     * If the user is an admin, flags to disable different checks
     */
    adminFlags?: {
        disableAllChecks?: boolean,
        disableInputValidationAndCasting?: boolean,
        disableMaxObjectsCheck?: boolean,
        disablePermissionsCheck?: boolean,
        disableProfanityCheck?: boolean,
        disableTriggerAfterMutations?: boolean,
    }
    info: PartialApiInfo,
    inputData: CudInputData[],
    userData: SessionUser,
}

type CudHelperResult = Array<boolean | Record<string, any>>;

type CudDataMaps = CudInputsToMapsResult;

type TopLevelInputsByAction = {
    Create: { index: number, input: PrismaCreate }[],
    Update: { index: number, input: PrismaUpdate }[],
    Delete: { index: number, input: string }[],
}
type TopLevelInputsByType = { [key in ModelType]?: TopLevelInputsByAction };

type CudTransactionData = {
    deletedIdsByType: { [key in ModelType]?: string[] };
    beforeDeletedData: { [key in ModelType]?: object };
    operations: PrismaPromise<object>[];
}

/**
 * Initializes the result array with false values.
 * @param length The length of the result array to initialize.
 * @returns An array filled with `false` of the specified length.
 */
function initializeResults(length: number): Array<boolean | Record<string, any>> {
    return new Array(length).fill(false);
}

/**
 * Validates and casts input data using the model's Yup schemas.
 * @param inputData The array of input data to validate and cast.
 * @throws CustomError if validation fails.
 */
async function validateAndCastInputs(inputData: CudInputData[]): Promise<void> {
    try {
        for (let i = 0; i < inputData.length; i++) {
            const { action, input, objectType } = inputData[i];
            const { mutate } = ModelMap.getLogic(["mutate"], objectType as ModelType, true, `cudHelper cast ${action.toLowerCase()}`);
            let transformedInput: AnyObjectSchema;
            if (action === "Create") {
                // If the mutate object doesn't have a yup.create property, it's not meant to be created this way
                if (!mutate.yup.create) {
                    throw new CustomError("0559", "InternalError", { action, objectType });
                }
                transformedInput = mutate.yup.create({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
            } else if (action === "Update") {
                // If the mutate object doesn't have a yup.update property, it's not meant to be updated this way
                if (!mutate.yup.update) {
                    throw new CustomError("0560", "InternalError", { action, objectType });
                }
                transformedInput = mutate.yup.update({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
            } else if (action === "Delete") {
                continue;
            } else {
                throw new CustomError("0558", "InternalError", { action });
            }
            inputData[i].input = transformedInput as unknown as PrismaCreate | PrismaUpdate;
        }
    } catch (error) {
        throw new CustomError("0561", "InternalError", { error, inputData });
    }
}

/**
 * Calculates pre-shape data for each type, which can include custom input validation.
 * @param inputsByType The inputs grouped by type.
 * @param userData The session user data.
 * @param inputsById The inputs grouped by ID.
 * @param preMap A map to store pre-shape data.
 */
export async function calculatePreShapeData(
    inputsByType: { [key: string]: any },
    userData: SessionUser,
    inputsById: { [key: string]: any },
    preMap: PreMap,
): Promise<void> {
    for (const [type, inputs] of Object.entries(inputsByType)) {
        preMap[type] = {};
        const { mutate } = ModelMap.getLogic(["mutate"], type as ModelType, true, "cudHelper preshape type");

        if (mutate.shape.pre) {
            const preResult = await mutate.shape.pre({ ...inputs, userData, inputsById });
            preMap[type] = preResult;
        }
    }
}

/**
 * Groups top-level input data (excluding relations) by type and action.
 * @param inputData The array of input data.
 * @returns An object grouping inputs by type and action.
 */
function groupTopLevelDataByType(inputData: CudInputData[]): TopLevelInputsByType {
    const topInputsByType: TopLevelInputsByType = {};

    for (const [index, { action, input, objectType }] of inputData.entries()) {
        if (!topInputsByType[objectType]) {
            topInputsByType[objectType] = { Create: [], Update: [], Delete: [] };
        }
        topInputsByType[objectType as ModelType]![action].push({ index, input: input as any });
    }

    return topInputsByType;
}

/**
 * Builds the array of database operations to be executed in the transaction.
 * @param topInputsByType The top-level inputs grouped by type and action.
 * @param inputData The original (but shaped) input data.
 * @param partialInfo The API endpoint info for selecting fields.
 * @param userData The session user data.
 * @param maps The maps containing grouped data.
 * @param additionalData Additional data to pass to the shape functions.
 * @returns An object containing the operations array and transaction data.
 */
async function buildOperations(
    topInputsByType: TopLevelInputsByType,
    inputData: CudInputData[],
    partialInfo: PartialApiInfo,
    userData: SessionUser,
    maps: CudDataMaps,
    additionalData: CudAdditionalData,
): Promise<CudTransactionData> {
    const operations: PrismaPromise<object>[] = [];
    const deletedIdsByType: { [key in ModelType]?: string[] } = {};
    const beforeDeletedData: { [key in ModelType]?: object } = {};

    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        const { dbTable, idField, mutate } = ModelMap.getLogic(["dbTable", "idField", "mutate"], objectType as ModelType, true, "cudHelper loop");
        const deletingIds = Delete.map(({ input }) => input);

        // Create operations
        for (const { index } of Create) {
            const { input } = inputData[index];
            // Make sure no objects with placeholder ids are created. These could potentially bypass permissions/api checks, 
            // since they're typically used to satisfy validation for relationships that aren't needed for the create 
            // (e.g. `listConnect` on a resource item that's already being created in a list)
            if ((input as PrismaCreate)?.id === DUMMY_ID) {
                throw new CustomError("0501", "InternalError", { input, objectType });
            }
            const data = mutate.shape.create
                ? await mutate.shape.create({
                    additionalData,
                    data: input,
                    idsCreateToConnect: maps.idsCreateToConnect,
                    preMap: maps.preMap,
                    userData
                })
                : input;
            const select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo)?.select;

            const createOperation = (DbProvider.get()[dbTable] as PrismaDelegate).create({
                data,
                select,
            }) as PrismaPromise<object>;
            operations.push(createOperation);
        }

        // Update operations
        for (const { index } of Update) {
            const { input } = inputData[index];

            // Check if the input has an ID and if the input is not empty
            if (!input || typeof input !== "object" || !(input as object)[idField]) {
                console.error(`[buildOperations] Missing required ID for update operation: ${JSON.stringify({ objectType, input, idField })}`);
                continue; // Skip this update operation
            }

            const data = mutate.shape.update
                ? await mutate.shape.update({
                    additionalData,
                    data: input,
                    idsCreateToConnect: maps.idsCreateToConnect,
                    preMap: maps.preMap,
                    userData
                })
                : input;

            // Verify that the shape function returned a valid update object
            if (!data || Object.keys(data).length === 0) {
                console.warn(`[buildOperations] Empty data after shape.update for ${objectType}:`, {
                    id: (input as PrismaUpdate)[idField],
                    originalInput: input,
                });
                continue; // Skip this update operation if data is empty
            }

            const select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo)?.select;

            const updateOperation = (DbProvider.get()[dbTable] as PrismaDelegate).update({
                where: { [idField]: (input as PrismaUpdate)[idField] },
                data,
                select,
            }) as PrismaPromise<object>;
            operations.push(updateOperation);
        }

        // Delete operations
        if (Delete.length > 0) {
            if (mutate.trigger?.beforeDeleted) {
                await mutate.trigger.beforeDeleted({ beforeDeletedData, deletingIds, userData });
            }

            try {
                const existingIds: string[] = await (DbProvider.get()[dbTable] as PrismaDelegate)
                    .findMany({
                        where: { [idField]: { in: deletingIds } },
                        select: { id: true },
                    })
                    .then((records) => records.map(({ id }) => id));

                deletedIdsByType[objectType as ModelType] = existingIds;

                const deleteOperation = (DbProvider.get()[dbTable] as PrismaDelegate).deleteMany({
                    where: { [idField]: { in: existingIds } },
                }) as PrismaPromise<object>;
                operations.push(deleteOperation);
            } catch (error) {
                throw new CustomError("0417", "InternalError", { error, deletingIds, objectType });
            }
        }
    }

    return {
        deletedIdsByType,
        beforeDeletedData,
        operations,
    };
}

/**
 * Executes all database operations in a transaction.
 * @param operations The array of database operation promises.
 * @returns The result of the transaction.
 * @throws CustomError if the transaction fails.
 */
async function executeTransaction(operations: PrismaPromise<object>[]): Promise<object[]> {
    try {
        return await DbProvider.get().$transaction(operations);
    } catch (error) {
        throw new CustomError("0557", "InternalError", { error });
    }
}

/**
 * Processes the results of the transaction and updates the result array.
 * @param transactionResult The result array from the transaction.
 * @param transactionData The data collected during the transaction preparation.
 * @param topInputsByType The top-level inputs grouped by type and action.
 * @param inputData The original input data.
 * @param partialInfo The partial GraphQL info for selecting fields.
 * @param maps The maps containing grouped data.
 * @param result The array to store the final results.
 */
function processTransactionResults(
    transactionResult: object[],
    transactionData: CudTransactionData,
    topInputsByType: TopLevelInputsByType,
    inputData: CudInputData[],
    partialInfo: PartialApiInfo,
    maps: CudDataMaps,
    result: Array<boolean | Record<string, any>>,
): void {
    let transactionIndex = 0;
    const { deletedIdsByType } = transactionData;

    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        for (const { index, input } of Create) {
            const createdObject = transactionResult[transactionIndex++];
            const converted = InfoConverter.get().fromDbToApi<object>(createdObject, partialInfo);
            result[index] = converted;
            if (input.id) {
                maps.resultsById[input.id as string] = converted;
            }
        }

        for (const { index, input } of Update) {
            const updatedObject = transactionResult[transactionIndex++];
            const converted = InfoConverter.get().fromDbToApi<object>(updatedObject, partialInfo);
            result[index] = converted;
            if (input.id) {
                maps.resultsById[input.id as string] = converted;
            }
        }

        if (Delete.length > 0) {
            const ids = deletedIdsByType[objectType as ModelType] || [];
            for (const id of ids) {
                const index = inputData.findIndex(({ input }) => input === id);
                if (index >= 0) result[index] = true;
            }
            transactionIndex++;
        }
    }
}

/**
 * Calls the afterMutations trigger for each type, if applicable.
 * @param transactionData The data collected during the transaction preparation.
 * @param maps The maps of data calculated by groupDataIntoMaps.
 * @param additionalData Additional data to pass to the trigger.
 * @param userData The session user data.
 */
async function triggerAfterMutations(
    transactionData: CudTransactionData,
    maps: CudDataMaps,
    additionalData: CudAdditionalData,
    userData: SessionUser,
): Promise<void> {
    const { deletedIdsByType, beforeDeletedData } = transactionData;
    const { idsByAction, inputsById, preMap, resultsById } = maps;
    const outputsByType = cudOutputsToMaps({ idsByAction, inputsById });
    const allTypes = new Set([...Object.keys(outputsByType), ...Object.keys(deletedIdsByType)]);

    for (const type of allTypes) {
        const { mutate } = ModelMap.getLogic(["mutate"], type as ModelType, true, "cudHelper afterMutations type");

        if (!mutate.trigger?.afterMutations) continue;

        const createInputs = outputsByType[type]?.createInputs || [];
        const createdIds = outputsByType[type]?.createdIds || [];
        const updatedIds = outputsByType[type]?.updatedIds || [];
        const updateInputs = outputsByType[type]?.updateInputs || [];
        const deletedIds = deletedIdsByType[type as ModelType] || [];

        await mutate.trigger.afterMutations({
            additionalData,
            beforeDeletedData,
            createdIds,
            createInputs,
            deletedIds,
            preMap,
            resultsById,
            updatedIds,
            updateInputs,
            userData,
        });
    }
}

/**
 * Performs create, update, and delete operations with validation, data shaping, and transaction management.
 * @returns An array of results corresponding to the inputData operations.
 */
export async function cudHelper({
    additionalData,
    adminFlags,
    info,
    inputData,
    userData,
}: CudHelperParams): Promise<CudHelperResult> {
    if (adminFlags && userData.id !== SEEDED_IDS.User.Admin) {
        throw new CustomError("0562", "Unauthorized", { adminFlags });
    }

    const result = initializeResults(inputData.length);
    if (!adminFlags?.disableInputValidationAndCasting && !adminFlags?.disableAllChecks) {
        await validateAndCastInputs(inputData);
    }

    const maps = await cudInputsToMaps({ inputData });
    await calculatePreShapeData(maps.inputsByType, userData, maps.inputsById, maps.preMap);

    const authDataById = await getAuthenticatedData(maps.idsByType, userData);
    if (!adminFlags?.disablePermissionsCheck && !adminFlags?.disableAllChecks) {
        await permissionsCheck(authDataById, maps.idsByAction, maps.inputsById, userData);
    }
    if (!adminFlags?.disableProfanityCheck && !adminFlags?.disableAllChecks) {
        profanityCheck(inputData, maps.inputsById, authDataById);
    }
    if (!adminFlags?.disableMaxObjectsCheck && !adminFlags?.disableAllChecks) {
        await maxObjectsCheck(maps.inputsById, authDataById, maps.idsByAction, userData);
    }

    const topInputsByType = groupTopLevelDataByType(inputData);

    const transactionData = await buildOperations(topInputsByType, inputData, info, userData, maps, additionalData || {});

    const transactionResult = await executeTransaction(transactionData.operations);
    processTransactionResults(transactionResult, transactionData, topInputsByType, inputData, info, maps, result);
    if (!adminFlags?.disableTriggerAfterMutations && !adminFlags?.disableAllChecks) {
        await triggerAfterMutations(transactionData, maps, additionalData || {}, userData);
    }
    return result;
}
