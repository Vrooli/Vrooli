import { Count, GqlModelType } from '@shared/consts';
import { cudInputsToMaps, getAuthenticatedData } from "../utils";
import { maxObjectsCheck, permissionsCheck, profanityCheck } from "../validators";
import { CUDHelperInput, CUDResult } from "./types";
import { modelToGraphQL, selectHelper } from "../builders";
import { getLogic } from "../getters";
import { reqArr } from '@shared/validation';

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
    console.log('cudhelper a', objectType);
    // Get functions for manipulating model logic
    const { delegate, mutate, validate } = getLogic(['delegate', 'mutate', 'validate'], objectType, userData.languages, 'cudHelper')
    // Initialize results
    let created: GqlModel[] = [], updated: GqlModel[] = [], deleted: Count = { __typename: 'Count' as const, count: 0 };
    // Initialize auth data by type
    let createAuthData: { [x: string]: any } = {}, updateAuthData: { [x: string]: any } = {};
    // Validate yup
    createMany && mutate.yup.create && reqArr(mutate.yup.create({})).validateSync(createMany, { abortEarly: false });
    updateMany && mutate.yup.update && reqArr(mutate.yup.update({})).validateSync(updateMany.map(u => u.data), { abortEarly: false });
    // Profanity check
    createMany && profanityCheck(createMany, partialInfo.__typename, userData.languages);
    updateMany && profanityCheck(updateMany.map(u => u.data), partialInfo.__typename, userData.languages);
    // Group create and update data by action and type
    console.log('cudhelper b');
    const { idsByAction, idsByType, inputsByType } = await cudInputsToMaps({
        createMany,
        updateMany,
        deleteMany,
        objectType,
        prisma,
        languages: userData.languages,
    });
    const preMap: { [x: string]: any } = {};
    console.log('cudhelper c', JSON.stringify(inputsByType), '\n\n');
    // For each type, calculate pre-shape data (if applicable)
    for (const [type, inputs] of Object.entries(inputsByType)) {
        // Initialize type as empty object
        preMap[type] = {};
        const { mutate } = getLogic(['mutate'], type as GqlModelType, userData.languages, 'preshape type');
        if (mutate.shape.pre) {
            console.log('getting pre shape', type)
            const { Create: createList, Update: updateList, Delete: deleteList } = inputs;
            const preResult = await mutate.shape.pre({ createList, updateList: updateList as any, deleteList, prisma, userData });
            console.log('got pre shape', type, JSON.stringify(preResult), '\n\n');
            preMap[type] = preResult;
        }
    }
    console.log('cudhelper d');
    // Shape create and update data. This must be done before other validations, as shaping may convert creates to connects
    const shapedCreate: { [x: string]: any }[] = [];
    const shapedUpdate: { where: { [x: string]: any }, data: { [x: string]: any } }[] = [];
    // Shape create
    if (createMany && mutate.shape.create) {
        for (const create of createMany) { shapedCreate.push(await mutate.shape.create({ data: create, preMap, prisma, userData })) }
    }
    console.log('cudhelper e');
    // Shape update
    if (updateMany && mutate.shape.update) {
        for (const update of updateMany) {
            const shaped = await mutate.shape.update({ data: update.data, preMap, prisma, userData, where: update.where as any });
            shapedUpdate.push({ where: update.where, data: shaped });
        }
    }
    console.log('cudhelper f');
    // Query for all authentication data
    const authDataById = await getAuthenticatedData(idsByType, prisma, userData);
    console.log('cudhelper g');
    // Validate permissions
    await permissionsCheck(authDataById, idsByAction, userData);
    console.log('cudhelper h');
    // Max objects check
    maxObjectsCheck(authDataById, idsByAction, prisma, userData);
    console.log('cudhelper i');
    // Perform custom validation for validations.common
    if (shapedCreate.length > 0 || shapedUpdate.length > 0 || (deleteMany && deleteMany.length > 0)) {
        console.log('cudhelper i.1', objectType, validate?.validations?.common?.toString())
        validate?.validations?.common && await validate.validations.common({
            connectMany: [],
            createMany: shapedCreate,
            deleteMany: deleteMany ?? [],
            disconnectMany: [],
            languages: userData.languages,
            prisma,
            updateMany: shapedUpdate,
            userData
        });
    }
    console.log('cudhelper j');
    if (shapedCreate.length > 0) {
        // Perform custom validation
        const deltaAdding = shapedCreate.length - (deleteMany ? deleteMany.length : 0);
        validate?.validations?.create && await validate.validations.create({ createMany: shapedCreate, languages: userData.languages, prisma, userData, deltaAdding });
        for (const data of shapedCreate) {
            // Create
            const select = await delegate(prisma).create({
                data,
                ...selectHelper(partialInfo)
            });
            // Convert
            console.log('created', JSON.stringify(select));
            const converted = modelToGraphQL<GqlModel>(select, partialInfo);
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
            userData
        });
    }
    console.log('cudhelper k');
    if (shapedUpdate.length > 0) {
        // Perform custom validation
        validate?.validations?.update && await validate.validations.update({ updateMany: shapedUpdate, languages: userData.languages, prisma, userData });
        for (const update of shapedUpdate) {
            // Update
            const select = await delegate(prisma).update({
                where: update.where,
                data: update.data,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL<GqlModel>(select, partialInfo);
            updated.push(converted as any);
        }
        // Filter authDataById to only include objects which were updated
        updateAuthData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => updated.map(u => u.id).includes(id)));
        // Call onUpdated
        mutate.trigger?.onUpdated && await mutate.trigger.onUpdated({
            authData: updateAuthData,
            preMap,
            prisma,
            updated,
            updateInput: updateMany!.map(u => u.data), userData
        });
    }
    console.log('cudhelper l');
    if (deleteMany && deleteMany.length > 0) {
        // Call beforeDeleted
        let beforeDeletedData: any;
        if (mutate.trigger?.beforeDeleted) {
            beforeDeletedData = await mutate.trigger.beforeDeleted({ deletingIds: deleteMany, prisma, userData });
        }
        // Perform custom validation
        validate?.validations?.delete && await validate.validations.delete({ deleteMany, languages: userData.languages, prisma, userData });
        deleted = await delegate(prisma).deleteMany({
            where: { id: { in: deleteMany } }
        }).then(({ count }) => ({ __typename: 'Count' as const, count }));
        // Call onDeleted
        mutate.trigger?.onDeleted && await mutate.trigger.onDeleted({
            beforeDeletedData,
            deleted,
            deletedIds: deleteMany,
            preMap,
            prisma,
            userData
        });
    }
    console.log('cudhelper m');
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
            userData
        });
    }
    console.log('cudhelper n');
    // For each type, calculate post-shape data (if applicable)
    for (const type in inputsByType) {
        const { mutate } = getLogic(['mutate'], type as GqlModelType, userData.languages, 'postshape type');
        if (mutate.shape.post) {
            await mutate.shape.post({
                created,
                deletedIds: deleteMany ?? [],
                prisma,
                updated,
                userData,
            });
        }
    }
    console.log('finished cudHelper');
    return {
        created: createMany ? created : undefined,
        updated: updateMany ? updated : undefined,
        deleted: deleteMany ? deleted : undefined,
    };
}