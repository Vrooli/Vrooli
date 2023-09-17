import { Count, DUMMY_ID, GqlModelType } from "@local/shared";
import { modelToGql, selectHelper } from "../builders";
import { PartialGraphQLInfo, PrismaCreate, PrismaUpdate } from "../builders/types";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { PrismaType, SessionUserToken } from "../types";
import { getAuthenticatedData } from "../utils";
import { cudInputsToMaps } from "../utils/cudInputsToMaps";
import { CudInputData } from "../utils/types";
import { maxObjectsCheck, permissionsCheck, profanityCheck } from "../validators";

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
export async function cudHelper<
    GqlModel extends ({ id: string } & { [x: string]: any })
>({
    inputData,
    partialInfo,
    prisma,
    userData,
}: {
    inputData: CudInputData[],
    partialInfo: PartialGraphQLInfo,
    prisma: PrismaType,
    userData: SessionUserToken,
}): Promise<Array<boolean | Record<string, any>>> {
    // Initialize results
    const result: Array<boolean | Record<string, any>> = new Array(inputData.length).fill(false);
    // Validate create and update data
    for (const { actionType, input, objectType } of inputData) {
        if (actionType === "Create") {
            const { mutate } = getLogic(["mutate"], objectType, userData.languages, "cudHelper create");
            mutate.yup.create && mutate.yup.create({}).validateSync(input);
        } else if (actionType === "Update") {
            const { mutate } = getLogic(["mutate"], objectType, userData.languages, "cudHelper update");
            mutate.yup.update && mutate.yup.update({}).validateSync(input);
        }
    }
    // Group all data, including relations, relations' relations, etc. into various maps. 
    // These are useful for validation and pre-shaping data
    console.time("cudInputsToMaps");
    const { idsByAction, idsByType, inputsById, inputsByType } = await cudInputsToMaps({
        inputData,
        prisma,
        languages: userData.languages,
    });
    console.timeEnd("cudInputsToMaps");
    const preMap: { [x: string]: any } = {};
    // For each type, calculate pre-shape data (if applicable). 
    // This often also doubles as a way to perform custom input validation
    for (const [type, inputs] of Object.entries(inputsByType)) {
        // Initialize type as empty object
        preMap[type] = {};
        const { mutate } = getLogic(["mutate"], type as GqlModelType, userData.languages, "preshape type");
        if (mutate.shape.pre) {
            const preResult = await mutate.shape.pre({ ...inputs, prisma, userData, inputsById });
            preMap[type] = preResult;
        }
    }
    // Query for all authentication data
    const authDataById = await getAuthenticatedData(idsByType, prisma, userData);
    // Validate permissions
    await permissionsCheck(authDataById, idsByAction, userData);
    // Perform profanity checks
    profanityCheck(inputData, authDataById, userData.languages);
    // Max objects check
    await maxObjectsCheck(authDataById, idsByAction, prisma, userData);
    // Group top-level (i.e. can't use inputsByType, idsByAction, etc. because those include relations) data by type
    const topInputsByType: { [key in GqlModelType]?: {
        Create: { index: number, input: PrismaCreate }[],
        Update: { index: number, input: PrismaUpdate }[],
        Delete: { index: number, input: string }[],
    } } = {};
    for (const [index, { actionType, input, objectType }] of inputData.entries()) {
        if (!topInputsByType[objectType]) {
            topInputsByType[objectType] = { Create: [], Update: [], Delete: [] };
        }
        topInputsByType[objectType]![actionType].push({ index, input: input as any });
    }
    // Loop through each type
    for (const [objectType, { Create, Update, Delete }] of Object.entries(topInputsByType)) {
        const { delegate, idField, mutate } = getLogic(["delegate", "idField", "mutate"], objectType as GqlModelType, userData.languages, "cudHelper.createOne");
        const created: any[] = [];
        const updateInputs: PrismaUpdate[] = [];
        const updated: any[] = [];
        const deletingIds = Delete.map(({ input }) => input);
        let deleted: Count = { __typename: "Count" as const, count: 0 };
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
                const data = mutate.shape.create ? await mutate.shape.create({ data: input, preMap, prisma, userData }) : input;
                // Create
                let createResult: object = {};
                let select: object | undefined;
                try {
                    select = selectHelper(partialInfo)?.select;
                    createResult = await delegate(prisma).create({
                        data,
                        select,
                    });
                } catch (error) {
                    throw new CustomError("0415", "InternalError", userData.languages, { error, data, select, objectType });
                }
                // Convert
                const converted = modelToGql<GqlModel>(createResult, partialInfo);
                result[index] = converted;
                created.push(converted);
            }
            // Call create trigger
            mutate.trigger?.onCreated && await mutate.trigger.onCreated({
                created,
                preMap,
                prisma,
                userData,
            });
        }
        // Update
        if (Update.length > 0) {
            for (const { index } of Update) {
                const { input, objectType } = inputData[index] as { input: PrismaUpdate, objectType: GqlModelType | `${GqlModelType}` };
                updateInputs.push(input);
                const data = mutate.shape.update ? await mutate.shape.update({ data: input, preMap, prisma, userData }) : input;
                // Update
                let updateResult: object = {};
                let select: object | undefined;
                try {
                    select = selectHelper(partialInfo)?.select;
                    updateResult = await delegate(prisma).update({
                        where: { [idField]: input[idField] },
                        data,
                        select,
                    });
                } catch (error) {
                    throw new CustomError("0416", "InternalError", userData.languages, { error, data, select, objectType });
                }
                // Convert
                const converted = modelToGql<GqlModel>(updateResult, partialInfo);
                result[index] = converted;
                updated.push(converted);
            }
            // Call update trigger
            mutate.trigger?.onUpdated && await mutate.trigger.onUpdated({
                preMap,
                prisma,
                updated,
                updateInputs,
                userData,
            });
        }
        // Delete
        if (Delete.length > 0) {
            // Call beforeDeleted
            let beforeDeletedData: object = [];
            if (mutate.trigger?.beforeDeleted) {
                beforeDeletedData = await mutate.trigger.beforeDeleted({ deletingIds, prisma, userData });
            }
            // Delete
            const where = { id: { in: deletingIds } };
            try {
                deleted = await delegate(prisma).deleteMany({ where }).then(({ count }) => ({ __typename: "Count" as const, count }));
                for (const { index } of Delete) { result[index] = true; }
            } catch (error) {
                throw new CustomError("0417", "InternalError", userData.languages, { error, where, objectType });
            }
            // Call onDeleted
            mutate.trigger?.onDeleted && await mutate.trigger.onDeleted({
                beforeDeletedData,
                deletedIds: deletingIds,
                preMap,
                prisma,
                userData,
            });
        }
        // Perform custom triggers for mutate.trigger.onCommon
        // NOTE: This is only for top-level objects, not relations
        if (Create.length > 0 || Update.length > 0 || Delete.length > 0) {
            mutate.trigger?.onCommon && await mutate.trigger.onCommon({
                created,
                deleted,
                deletedIds: deletingIds,
                preMap,
                prisma,
                updated,
                updateInputs,
                userData,
            });
        }
    }
    // For each type (including relations), calculate post-shape data (e.g. updating indexes)
    // TODO need to somehow get created, updated, and deleted info of relations
    // TODO for when I start looking at this again: Should get rid of triggers altogether, in favor of pre and post. 
    // If there are any situations that specifically required a trigger before (e.g. if role creation could also create an organization 
    // instead of connecting), then we can use the tree node to check if parent is "Admin" role before creating the "Admin" role. But 
    // notice how this even isn't a real example. I'm not sure if triggering something only on the top-level object is even useful anymore.
    for (const type in inputsByType) {
        const { mutate } = getLogic(["mutate"], type as GqlModelType, userData.languages, "postshape type");
        if (mutate.shape.post) {
            await mutate.shape.post({
                created,
                deletedIds: deleteMany ?? [],
                preMap,
                prisma,
                updated,
                userData,
            });
        }
    }
    return result;
}
