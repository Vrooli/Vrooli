import { DUMMY_ID, generatePK, ModelType, SessionUser, validatePK, YupMutateParams } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { AnyObjectSchema, ValidationError as YupError } from "yup";
import { InfoConverter } from "../builders/infoConverter.js";
import { PartialApiInfo, PrismaCreate, PrismaDelegate, PrismaUpdate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";
import { PreMap } from "../models/types.js";
import { cudInputsToMaps, CudInputsToMapsResult } from "../utils/cudInputsToMaps.js";
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

interface FieldError {
    path: string;              // e.g.  "email"  or "address[0].zip"
    message: string;           // human‑readable, from Yup
}

interface ValidationErrorPayload {
    objectType: string;
    action: string;
    errors: FieldError[];
}

/**
 * Validates and casts input data using the model's Yup schemas.
 * @param inputData The array of input data to validate and cast.
 * @throws CustomError if validation fails.
 */
async function validateAndCastInputs(inputData: CudInputData[]): Promise<void> {
    for (let i = 0; i < inputData.length; i++) {
        const { action, input, objectType } = inputData[i];
        const { mutate } = ModelMap.getLogic(["mutate"], objectType as ModelType, true, "cudHelper cast");

        const schemaFactory:
            | ((p: YupMutateParams) => AnyObjectSchema)
            | undefined =
            action === "Create"
                ? mutate.yup.create
                : action === "Update"
                    ? mutate.yup.update
                    : undefined;

        if (!schemaFactory) {
            throw new CustomError("0953", "InternalError", { action, objectType });
        }

        try {
            const transformed = schemaFactory({ env: process.env.NODE_ENV }).cast(input, { stripUnknown: true });
            inputData[i].input = transformed as PrismaCreate | PrismaUpdate;

        } catch (err) {
            if (err instanceof YupError) {
                const fieldErrors: FieldError[] = err.inner.length
                    ? err.inner.map(e => ({ path: e.path ?? "", message: e.message }))
                    : [{ path: err.path ?? "", message: err.message }];

                const payload: ValidationErrorPayload = { objectType, action, errors: fieldErrors };
                throw new CustomError("0558", "ValidationFailed", payload);
            }
            throw new CustomError("0959", "InternalError", { err, objectType });
        }
    }
}

/**
 * Replaces every non‑Snowflake id in CREATE inputs (and every reference that
 * points at them) with a freshly‑generated Snowflake.  
 * Returns the placeholder ➜ snowflake map in case callers want it.
 */
export function convertCreateIds(inputData: CudInputData[]) {
    /** placeholder → snowflake */
    const map = new Map<string, bigint>();

    /** Recursive walker that rewrites any scalar or scalar‑inside‑array/object */
    function deepReplace(value: any): any {
        // scalar → swap if we have a mapping
        if (typeof value === "string" || typeof value === "number") {
            const hit = map.get(String(value));
            return hit ? hit.toString() : value;
        }

        // array → recurse on every element
        if (Array.isArray(value)) {
            return value.map(deepReplace);
        }

        // object → recurse on every key
        if (value && typeof value === "object") {
            for (const key of Object.keys(value)) {
                value[key] = deepReplace(value[key]);
            }
        }

        return value;
    }

    for (const entry of inputData) {
        if (entry.action !== "Create") continue;

        const create = entry.input as PrismaCreate;
        const currentId = (create as any).id ?? DUMMY_ID;

        if (!validatePK(currentId)) {
            const sf = generatePK();
            map.set(String(currentId), sf);
            create.id = sf.toString();
        }
    }

    for (const entry of inputData) {
        entry.input = deepReplace(entry.input);
    }

    return map;
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

        // Calculate select object once per type, default if empty
        let select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo)?.select;
        if (!select || Object.keys(select).length === 0) {
            logger.warning(`[buildOperations] Empty select for ${objectType}. Defaulting to { id: true }.`, { trace: "0800" });
            select = { id: true };
        }

        // Create operations
        for (const { index } of Create) {
            const { input } = inputData[index];
            const data = mutate.shape.create
                ? await mutate.shape.create({
                    additionalData,
                    data: input,
                    idsCreateToConnect: maps.idsCreateToConnect,
                    preMap: maps.preMap,
                    userData,
                })
                : input;

            const createOperation = (DbProvider.get()[dbTable] as PrismaDelegate).create({
                data,
                select,
            }) as PrismaPromise<object>;
            operations.push(createOperation);
        }

        // Update operations
        for (const { index } of Update) {
            const { input } = inputData[index];

            const data = mutate.shape.update
                ? await mutate.shape.update({
                    additionalData,
                    data: input,
                    idsCreateToConnect: maps.idsCreateToConnect,
                    preMap: maps.preMap,
                    userData,
                })
                : input;

            // Verify that the shape function returned a valid update object
            if (!data || Object.keys(data).length === 0) {
                logger.warning(`[buildOperations] Empty data after shape.update for ${objectType}`, { trace: "0801" });
                continue; // Skip this update operation if data is empty
            }

            // Verify that the "where" object is valid
            const where = { id: typeof input === "string" ? input : input.id };
            if (typeof where.id !== "string") {
                logger.error(`[buildOperations] Invalid where object for update operation: ${JSON.stringify({ objectType, input, where })}`, { trace: "0802" });
                throw new CustomError("0802", "InternalError", { objectType });
            }

            const updateOperation = (DbProvider.get()[dbTable] as PrismaDelegate).update({
                where,
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
    const adminId = await DbProvider.getAdminId();
    if (adminFlags && userData.id !== adminId) {
        throw new CustomError("0562", "Unauthorized", { adminFlags });
    }

    const result = new Array(inputData.length).fill(false);
    if (!adminFlags?.disableInputValidationAndCasting && !adminFlags?.disableAllChecks) {
        await validateAndCastInputs(inputData);
    }

    convertCreateIds(inputData);

    const maps = await cudInputsToMaps({ inputData });
    Object.freeze(maps);
    await calculatePreShapeData(maps.inputsByType, userData, maps.inputsById, maps.preMap);
    Object.freeze(maps.idsByAction);
    Object.freeze(maps.idsByType);
    Object.freeze(maps.idsCreateToConnect);
    Object.freeze(maps.inputsById);
    Object.freeze(maps.inputsByType);
    Object.freeze(maps.preMap);
    const authDataById = await getAuthenticatedData(maps.idsByType, userData);
    Object.freeze(authDataById);

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
    Object.freeze(topInputsByType);

    const transactionData = await buildOperations(topInputsByType, inputData, info, userData, maps, additionalData || {});
    Object.freeze(transactionData);

    const transactionResult = await executeTransaction(transactionData.operations);
    Object.freeze(transactionResult);
    processTransactionResults(transactionResult, transactionData, topInputsByType, inputData, info, maps, result);
    Object.freeze(maps.resultsById);

    if (!adminFlags?.disableTriggerAfterMutations && !adminFlags?.disableAllChecks) {
        await triggerAfterMutations(transactionData, maps, additionalData || {}, userData);
    }
    return result;
}
