import { Count, DUMMY_ID, GqlModelType, reqArr } from "@local/shared";
import { modelToGql, selectHelper } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { cudInputsToMaps, getAuthenticatedData } from "../utils";
import { maxObjectsCheck, permissionsCheck, profanityCheck } from "../validators";
import { CUDHelperInput, CUDResult } from "./types";

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
 * Finally, it performs the operations in the database, and returns the results in shape of GraphQL objects
 */
export async function cudHelper<
    GqlModel extends ({ id: string } & { [x: string]: any })
>({
    createMany,
    deleteMany,
    objectType,
    partialInfo,
    prisma,
    updateMany,
    userData,
}: CUDHelperInput): Promise<CUDResult<GqlModel>> {
    // Get functions for manipulating model logic
    const { delegate, mutate } = getLogic(["delegate", "mutate", "validate"], objectType, userData.languages, "cudHelper");
    // Initialize results
    const created: GqlModel[] = [], updated: GqlModel[] = [];
    let deleted: Count = { __typename: "Count" as const, count: 0 };
    // Initialize auth data by type
    let createAuthData: { [x: string]: any } = {}, updateAuthData: { [x: string]: any } = {};
    // Validate yup
    createMany && mutate.yup.create && reqArr(mutate.yup.create({})).validateSync(createMany, { abortEarly: false });
    updateMany && mutate.yup.update && reqArr(mutate.yup.update({})).validateSync(updateMany.map(u => u.data), { abortEarly: false });
    // Profanity check
    createMany && profanityCheck(createMany, partialInfo.__typename, userData.languages);
    updateMany && profanityCheck(updateMany.map(u => u.data), partialInfo.__typename, userData.languages);
    // Group create and update data by action and type
    const { idsByAction, idsByType, inputsByType } = await cudInputsToMaps({
        createMany,
        updateMany,
        deleteMany,
        objectType,
        prisma,
        languages: userData.languages,
    });
    const preMap: { [x: string]: any } = {};
    // For each type, calculate pre-shape data (if applicable). 
    // This often also doubles as a way to perform custom input validation
    for (const [type, inputs] of Object.entries(inputsByType)) {
        // Initialize type as empty object
        preMap[type] = {};
        const { mutate } = getLogic(["mutate"], type as GqlModelType, userData.languages, "preshape type");
        if (mutate.shape.pre) {
            const { Create: createList, Update: updateList, Delete: deleteList } = inputs;
            const preResult = await mutate.shape.pre({ createList, updateList: updateList as any, deleteList, prisma, userData });
            preMap[type] = preResult;
        }
    }
    // Shape create and update data. This must be done before other validations, as shaping may convert creates to connects
    const shapedCreate: { [x: string]: any }[] = [];
    const shapedUpdate: { where: { [x: string]: any }, data: { [x: string]: any } }[] = [];
    // Shape create
    if (createMany && mutate.shape.create) {
        for (const create of createMany) { shapedCreate.push(await mutate.shape.create({ data: create, preMap, prisma, userData })); }
    }
    // Shape update
    if (updateMany && mutate.shape.update) {
        for (const update of updateMany) {
            const shaped = await mutate.shape.update({ data: update.data, preMap, prisma, userData, where: update.where as any });
            shapedUpdate.push({ where: update.where, data: shaped });
        }
    }
    // Query for all authentication data
    const authDataById = await getAuthenticatedData(idsByType, prisma, userData);
    // Validate permissions
    await permissionsCheck(authDataById, idsByAction, userData);
    // Max objects check
    maxObjectsCheck(authDataById, idsByAction, prisma, userData);
    if (shapedCreate.length > 0) {
        for (const data of shapedCreate) {
            // Make sure no objects with placeholder ids are created. These could potentially bypass permissions/api checks, 
            // since they're typically used to satisfy validation for relationships that aren't needed for the create 
            // (e.g. `listConnect` on a resource item that's already being created in a list)
            if (data?.id === DUMMY_ID) {
                throw new CustomError("0501", "InternalError", userData.languages, { data, objectType });
            }
            // Create
            let createResult: any = {};
            let select: { [key: string]: any } | undefined;
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
            created.push(converted as any);
        }
        // Filter authDataById to only include objects which were created
        createAuthData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => created.map(c => c.id).includes(id)));
        // Call onCreated
        mutate.trigger?.onCreated && await mutate.trigger.onCreated({
            authData: createAuthData,
            created,
            preMap,
            prisma,
            userData,
        });
    }
    if (shapedUpdate.length > 0 && updateMany) {
        for (const update of shapedUpdate) {
            // Update
            let updateResult: object = {};
            let select: object | undefined;
            try {
                select = selectHelper(partialInfo)?.select;
                updateResult = await delegate(prisma).update({
                    where: update.where,
                    data: update.data,
                    select,
                });
            } catch (error) {
                throw new CustomError("0416", "InternalError", userData.languages, { error, update, select, objectType });
            }
            // Convert
            const converted = modelToGql<GqlModel>(updateResult, partialInfo);
            updated.push(converted as GqlModel);
        }
        // Filter authDataById to only include objects which were updated
        updateAuthData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => updated.map(u => u.id).includes(id)));
        // Call onUpdated
        mutate.trigger?.onUpdated && await mutate.trigger.onUpdated({
            authData: updateAuthData,
            preMap,
            prisma,
            updated,
            updateInput: updateMany.map(u => u.data), userData,
        });
    }
    if (deleteMany && deleteMany.length > 0) {
        // Call beforeDeleted
        let beforeDeletedData: object = [];
        if (mutate.trigger?.beforeDeleted) {
            beforeDeletedData = await mutate.trigger.beforeDeleted({ deletingIds: deleteMany, prisma, userData });
        }
        // Delete
        const where = { id: { in: deleteMany } };
        try {
            deleted = await delegate(prisma).deleteMany({
                where,
            }).then(({ count }) => ({ __typename: "Count" as const, count }));
        } catch (error) {
            throw new CustomError("0417", "InternalError", userData.languages, { error, where, objectType });
        }
        // Call onDeleted
        mutate.trigger?.onDeleted && await mutate.trigger.onDeleted({
            beforeDeletedData,
            deleted,
            deletedIds: deleteMany,
            preMap,
            prisma,
            userData,
        });
    }
    // Perform custom triggers for mutate.trigger.onCommon
    if (shapedCreate.length > 0 || shapedUpdate.length > 0 || (deleteMany && deleteMany.length > 0)) {
        mutate.trigger?.onCommon && await mutate.trigger.onCommon({
            createAuthData,
            created,
            deleted,
            deletedIds: deleteMany ?? [],
            preMap,
            prisma,
            updateAuthData,
            updated,
            updateInput: updateMany?.map(u => u.data) ?? [],
            userData,
        });
    }
    // For each type, calculate post-shape data (if applicable)
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
    return {
        created: createMany ? created : undefined,
        updated: updateMany ? updated : undefined,
        deleted: deleteMany ? deleted : undefined,
    };
}
