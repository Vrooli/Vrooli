import { DUMMY_ID, GqlModelType } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { modelToGql } from "../builders/modelToGql";
import { selectHelper } from "../builders/selectHelper";
import { PartialGraphQLInfo, PrismaCreate, PrismaDelegate, PrismaUpdate } from "../builders/types";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { PreMap } from "../models/types";
import { SessionUserToken } from "../types";
import { CudInputsToMapsResult, cudInputsToMaps } from "../utils/cudInputsToMaps";
import { cudOutputsToMaps } from "../utils/cudOutputsToMaps";
import { getAuthenticatedData } from "../utils/getAuthenticatedData";
import { CudInputData } from "../utils/types";
import { maxObjectsCheck } from "../validators/maxObjectsCheck";
import { permissionsCheck } from "../validators/permissions";
import { profanityCheck } from "../validators/profanityCheck";
import { CudAdditionalData } from "./types";

type CudHelperParams = {
    inputData: CudInputData[],
    partialInfo: PartialGraphQLInfo,
    userData: SessionUserToken,
    /** Additional data that can be passed to ModelLogic functions */
    additionalData?: CudAdditionalData,
}

type CudHelperResult = Array<boolean | Record<string, any>>;

type CudDataMaps = CudInputsToMapsResult;

type TopLevelInputsByAction = {
    Create: { index: number, input: PrismaCreate }[],
    Update: { index: number, input: PrismaUpdate }[],
    Delete: { index: number, input: string }[],
}
type TopLevelInputsByType = { [key in GqlModelType]?: TopLevelInputsByAction };

type CudTransactionData = {
    deletedIdsByType: { [key in GqlModelType]?: string[] };
    beforeDeletedData: { [key in GqlModelType]?: object };
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
 * @param userData The session user data.
 * @throws CustomError if validation fails.
 */
async function validateAndCastInputs(
    inputData: CudInputData[],
    userData: SessionUserToken
): Promise<void> {
    for (let i = 0; i < inputData.length; i++) {
        const { action, input, objectType } = inputData[i];
        const { mutate } = ModelMap.getLogic(["mutate"], objectType as GqlModelType, true, `cudHelper cast ${action.toLowerCase()}`);
        let transformedInput;
        if (action === "Create") {
            if (!mutate.yup.create) {
                throw new CustomError("0559", "InternalError", userData.languages, { action, objectType });
            }
            transformedInput = mutate.yup.create({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
        } else if (action === "Update") {
            if (!mutate.yup.update) {
                throw new CustomError("0560", "InternalError", userData.languages, { action, objectType });
            }
            transformedInput = mutate.yup.update({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
        } else if (action === "Delete") {
            continue;
        } else {
            throw new CustomError("0558", "InternalError", userData.languages, { action });
        }
        inputData[i].input = transformedInput;
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
    userData: SessionUserToken,
    inputsById: { [key: string]: any },
    preMap: PreMap
): Promise<void> {
    for (const [type, inputs] of Object.entries(inputsByType)) {
        preMap[type] = {};
        const { mutate } = ModelMap.getLogic(["mutate"], type as GqlModelType, true, "cudHelper preshape type");

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
        topInputsByType[objectType as GqlModelType]![action].push({ index, input: input as any });
    }

    return topInputsByType;
}

/**
 * Builds the array of database operations to be executed in the transaction.
 * @param topInputsByType The top-level inputs grouped by type and action.
 * @param inputData The original (but shaped) input data.
 * @param partialInfo The partial GraphQL info for selecting fields.
 * @param userData The session user data.
 * @param maps The maps containing grouped data.
 * @returns An object containing the operations array and transaction data.
 */
async function buildOperations(
    topInputsByType: TopLevelInputsByType,
    inputData: CudInputData[],
    partialInfo: PartialGraphQLInfo,
    userData: SessionUserToken,
    maps: CudDataMaps,
): Promise<CudTransactionData> {
    const operations: PrismaPromise<object>[] = [];
    const deletedIdsByType: { [key in GqlModelType]?: string[] } = {};
    const beforeDeletedData: { [key in GqlModelType]?: object } = {};

    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        const { dbTable, idField, mutate } = ModelMap.getLogic(["dbTable", "idField", "mutate"], objectType as GqlModelType, true, "cudHelper loop");
        const deletingIds = Delete.map(({ input }) => input);

        // Create operations
        for (const { index } of Create) {
            const { input } = inputData[index];
            // Make sure no objects with placeholder ids are created. These could potentially bypass permissions/api checks, 
            // since they're typically used to satisfy validation for relationships that aren't needed for the create 
            // (e.g. `listConnect` on a resource item that's already being created in a list)
            if ((input as PrismaCreate)?.id === DUMMY_ID) {
                throw new CustomError("0501", "InternalError", userData.languages, { input, objectType });
            }
            const data = mutate.shape.create
                ? await mutate.shape.create({ data: input, idsCreateToConnect: maps.idsCreateToConnect, preMap: maps.preMap, userData })
                : input;
            const select = selectHelper(partialInfo)?.select;

            const createOperation = (prismaInstance[dbTable] as PrismaDelegate).create({
                data,
                select,
            }) as PrismaPromise<object>;
            operations.push(createOperation);
        }

        // Update operations
        for (const { index } of Update) {
            const { input } = inputData[index];
            const data = mutate.shape.update
                ? await mutate.shape.update({ data: input, idsCreateToConnect: maps.idsCreateToConnect, preMap: maps.preMap, userData })
                : input;
            const select = selectHelper(partialInfo)?.select;

            const updateOperation = (prismaInstance[dbTable] as PrismaDelegate).update({
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
                const existingIds: string[] = await (prismaInstance[dbTable] as PrismaDelegate)
                    .findMany({
                        where: { [idField]: { in: deletingIds } },
                        select: { id: true },
                    })
                    .then((records) => records.map(({ id }) => id));

                deletedIdsByType[objectType as GqlModelType] = existingIds;

                const deleteOperation = (prismaInstance[dbTable] as PrismaDelegate).deleteMany({
                    where: { [idField]: { in: existingIds } },
                }) as PrismaPromise<object>;
                operations.push(deleteOperation);
            } catch (error) {
                throw new CustomError("0417", "InternalError", userData.languages, { error, deletingIds, objectType });
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
 * @param userData The session user data.
 * @returns The result of the transaction.
 * @throws CustomError if the transaction fails.
 */
async function executeTransaction(operations: PrismaPromise<object>[], userData: SessionUserToken): Promise<object[]> {
    try {
        return await prismaInstance.$transaction(operations);
    } catch (error) {
        throw new CustomError("0557", "InternalError", userData.languages, { error });
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
    partialInfo: PartialGraphQLInfo,
    maps: CudDataMaps,
    result: Array<boolean | Record<string, any>>
): void {
    let transactionIndex = 0;
    const { deletedIdsByType } = transactionData;

    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        for (const { index, input } of Create) {
            const createdObject = transactionResult[transactionIndex++];
            const converted = modelToGql<object>(createdObject, partialInfo);
            result[index] = converted;
            if (input.id) {
                maps.resultsById[input.id as string] = converted;
            }
        }

        for (const { index, input } of Update) {
            const updatedObject = transactionResult[transactionIndex++];
            const converted = modelToGql<object>(updatedObject, partialInfo);
            result[index] = converted;
            if (input.id) {
                maps.resultsById[input.id as string] = converted;
            }
        }

        if (Delete.length > 0) {
            const ids = deletedIdsByType[objectType as GqlModelType] || [];
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
    userData: SessionUserToken,
): Promise<void> {
    const { deletedIdsByType, beforeDeletedData } = transactionData;
    const { idsByAction, inputsById, preMap, resultsById } = maps;
    const outputsByType = cudOutputsToMaps({ idsByAction, inputsById });
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
 * @param params An object containing inputData, partialInfo, and userData.
 * @returns An array of results corresponding to the inputData operations.
 */
export async function cudHelper(params: CudHelperParams): Promise<CudHelperResult> {
    const { additionalData, inputData, partialInfo, userData } = params;

    const result = initializeResults(inputData.length);
    await validateAndCastInputs(inputData, userData);

    const maps = await cudInputsToMaps({ inputData })
    await calculatePreShapeData(maps.inputsByType, userData, maps.inputsById, maps.preMap);

    const authDataById = await getAuthenticatedData(maps.idsByType, userData);
    await permissionsCheck(authDataById, maps.idsByAction, maps.inputsById, userData);
    profanityCheck(inputData, maps.inputsById, authDataById, userData.languages);
    await maxObjectsCheck(maps.inputsById, authDataById, maps.idsByAction, userData);

    const topInputsByType = groupTopLevelDataByType(inputData);

    const transactionData = await buildOperations(topInputsByType, inputData, partialInfo, userData, maps);

    const transactionResult = await executeTransaction(transactionData.operations, userData);
    processTransactionResults(transactionResult, transactionData, topInputsByType, inputData, partialInfo, maps, result);
    await triggerAfterMutations(transactionData, maps, additionalData || {}, userData);

    return result;
}