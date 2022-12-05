import { Count } from "../endpoints/types";
import { Mutater } from "../models/types";
import { getAuthenticatedData, getAuthenticatedIds } from "../utils";
import { maxObjectsCheck, permissionsCheck, profanityCheck } from "../validators";
import { CUDHelperInput, CUDResult } from "./types";
import { getDelegator, getMutater, getValidator } from "../getters";
import { modelToGraphQL, selectHelper } from "../builders";

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
    GQLObject extends ({ id: string} & { [x: string]: any })
>({
    createMany,
    deleteMany,
    objectType,
    partialInfo,
    prisma,
    updateMany,
    userData,
}: CUDHelperInput): Promise<CUDResult<GQLObject>> {
    // Get mutater
    const mutater: Mutater<GQLObject, { graphql: any, db: any }, { graphql: any, db: any }, any, any> = getMutater(objectType, userData.languages, 'cudHelper');
    // Initialize results
    let created: GQLObject[] = [], updated: GQLObject[] = [], deleted: Count = { count: 0 };
    // Validate yup
    createMany && mutater.yup.create.validateSync(createMany, { abortEarly: false });
    updateMany && mutater.yup.update.validateSync(updateMany.map(u => u.data), { abortEarly: false });
    // Profanity check
    createMany && profanityCheck(createMany, partialInfo.__typename, userData.languages);
    updateMany && profanityCheck(updateMany.map(u => u.data), partialInfo.__typename, userData.languages);
    // Shape create and update data. This must be done before other validations, as shaping may convert creates to connects
    const shapedCreate: { [x: string]: any }[] = [];
    const shapedUpdate: { where: { [x: string]: any }, data: { [x: string]: any } }[] = [];
    if (createMany) {
        for (const create of createMany) { shapedCreate.push(await mutater.shape.create({ data: create, prisma, userData })) }
    }
    if (updateMany) {
        for (const update of updateMany) {
            const shaped = await mutater.shape.update({ data: update.data, prisma, userData, where: update.where });
            shapedUpdate.push({ where: update.where, data: shaped });
        }
    }
    // Find validator and prisma delegate for this object type
    const validator = getValidator(objectType, userData.languages, 'cudHelper');
    const prismaDelegate = getDelegator(objectType, prisma, userData.languages, 'cudHelper');
    // Get IDs of all objects which need to be authenticated
    const { idsByType, idsByAction } = await getAuthenticatedIds([
        ...(shapedCreate.map(data => ({ actionType: 'Create', data }))),
        ...(shapedUpdate.map(data => ({ actionType: 'Update', data: data.data }))),
        ...((deleteMany || [] as any).map(id => ({ actionType: 'Delete', id }))),
    ], objectType, prisma, userData.languages)
    // Query for all authentication data
    const authDataById = await getAuthenticatedData(idsByType, prisma, userData);
    // Validate permissions
    permissionsCheck(authDataById, idsByAction, userData);
    console.log("finished permissions check");
    // Max objects check
    maxObjectsCheck(authDataById, idsByAction, prisma, userData);
    console.log("finished max objects check");
    if (shapedCreate.length > 0) {
        // Perform custom validation
        const deltaAdding = shapedCreate.length - (deleteMany ? deleteMany.length : 0);
        console.log('going to validate create')
        validator?.validations?.create && await validator.validations.create({ createMany: shapedCreate, languages: userData.languages, prisma, userData, deltaAdding });
        console.log('validated create');
        for (const data of shapedCreate) {
            // Create
            const select = await prismaDelegate.create({
                data,
                ...selectHelper(partialInfo)
            });
            // Convert
            console.log('created', JSON.stringify(select));
            const converted = modelToGraphQL<GQLObject>(select, partialInfo);
            created.push(converted as any);
        }
        // Filter authDataById to only include objects which were created
        const authData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => created.map(c => c.id).includes(id)));
        // Call onCreated
        mutater.trigger?.onCreated && await mutater.trigger.onCreated({ authData, created, prisma, userData });
    }
    if (shapedUpdate.length > 0) {
        // Perform custom validation
        validator?.validations?.update && await validator.validations.update({ updateMany: shapedUpdate.map(u => u.data), languages: userData.languages, prisma, userData });
        for (const update of shapedUpdate) {
            // Update
            const select = await prismaDelegate.update({
                where: update.where,
                data: update.data,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL<GQLObject>(select, partialInfo);
            updated.push(converted as any);
        }
        // Filter authDataById to only include objects which were updated
        const authData = Object.fromEntries(Object.entries(authDataById).filter(([id]) => updated.map(u => u.id).includes(id)));
        // Call onUpdated
        mutater.trigger?.onUpdated && await mutater.trigger.onUpdated({ authData, prisma, updated, updateInput: updateMany!.map(u => u.data), userData });
    }
    if (deleteMany) {
        // Perform custom validation
        validator?.validations?.delete && await validator.validations.delete({ deleteMany, languages: userData.languages, prisma, userData });
        deleted = await prismaDelegate.deleteMany({
            where: { id: { in: deleteMany } }
        })
        // Call onDeleted
        mutater.trigger?.onDeleted && await mutater.trigger.onDeleted({ deleted, prisma, userData });
    }
    console.log('finished cudHelper');
    return {
        created: createMany ? created : undefined,
        updated: updateMany ? updated : undefined,
        deleted: deleteMany ? deleted : undefined,
    };
}