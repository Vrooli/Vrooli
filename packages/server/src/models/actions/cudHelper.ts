import { Count } from "../../schema/types";
import { modelToGraphQL, selectHelper } from "../builder";
import { CUDHelperInput, CUDResult } from "../types";
import { getAuthenticatedData, getAuthenticatedIds, getDelegate, getValidator } from "../utils";
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
 * Finally, it performs the operations in the database, and returns the results in shape of GraphQL objects
 */
export async function cudHelper<GQLCreate extends { [x: string]: any }, GQLUpdate extends { [x: string]: any }, GQLObject extends { [x: string]: any }, DBCreate extends { [x: string]: any }, DBUpdate extends { [x: string]: any }>({
    createMany,
    deleteMany,
    objectType,
    onCreated,
    onUpdated,
    onDeleted,
    partialInfo,
    prisma,
    shape,
    updateMany,
    userData,
    yup,
}: CUDHelperInput<GQLCreate, GQLUpdate, GQLObject, DBCreate, DBUpdate>): Promise<CUDResult<GQLObject>> {
    // Initialize results
    let created: GQLObject[] = [], updated: GQLObject[] = [], deleted: Count = { count: 0 };
    // Validate yup
    createMany && yup.yupCreate.validateSync(createMany, { abortEarly: false });
    updateMany && yup.yupUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
    // Profanity check
    createMany && profanityCheck(createMany, partialInfo);
    updateMany && profanityCheck(updateMany.map(u => u.data), partialInfo);
    // Shape create and update data. This must be done before other validations, as shaping may convert creates to connects
    const shapedCreate: DBCreate[] = [];
    const shapedUpdate: { where: { [x: string]: any }, data: DBUpdate }[] = [];
    if (createMany) {
        for (const create of createMany) { shapedCreate.push(await shape.shapeCreate(userId, create)) }
    }
    if (updateMany) {
        for (const update of updateMany) {
            const shaped = await shape.shapeUpdate(userId, update.data);
            shapedUpdate.push({ where: update.where, data: shaped });
        }
    }
    // Find validator and prisma delegate for this object type
    const validator = getValidator(objectType, 'cudHelper');
    const prismaDelegate = getDelegate(objectType, prisma, 'cudHelper');
    // Get IDs of all objects which need to be authenticated
    const { idsByType, idsByAction } = await getAuthenticatedIds([
        ...(shapedCreate.map(data => ({ actionType: 'Create', data }))),
        ...(shapedUpdate.map(data => ({ actionType: 'Update', data: data.data }))),
        ...((deleteMany || [] as any).map(id => ({ actionType: 'Delete', id }))),
    ], objectType, prisma)
    // Query for all authentication data
    const authDataById = await getAuthenticatedData(idsByType, prisma, userId);
    // Validate permissions
    permissionsCheck(authDataById, idsByAction, userId);
    // Max objects check
    maxObjectsCheck(authDataById, idsByAction, prisma, userId);
    if (shapedCreate.length > 0) {
        // Perform custom validation
        const deltaAdding = shapedCreate.length - (deleteMany ? deleteMany.length : 0);
        validator?.validations?.create && await validator.validations.create(shapedCreate, prisma, userId, deltaAdding);
        for (const data of shapedCreate) {
            // Create
            const select = await prismaDelegate.create({
                data,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL<GQLObject>(select, partialInfo);
            created.push(converted);
        }
        // Call onCreated
        onCreated && await onCreated(created);
    }
    if (shapedUpdate.length > 0) {
        // Perform custom validation
        validator?.validations?.update && await validator.validations.update(shapedUpdate.map(u => u.data), prisma, userId);
        for (const update of shapedUpdate) {
            // Update
            const select = await prismaDelegate.update({
                where: update.where,
                data: update.data,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL<GQLObject>(select, partialInfo);
            updated.push(converted);
        }
        // Call onUpdated
        onUpdated && await onUpdated(updated, updateMany!.map(u => u.data));
    }
    if (deleteMany) {
        // Perform custom validation
        validator?.validations?.delete && await validator.validations.delete(deleteMany, prisma, userId);
        deleted = await prismaDelegate.deleteMany({
            where: { id: { in: deleteMany } }
        })
        // Call onDeleted
        onDeleted && await onDeleted(deleted);
    }
    return {
        created: createMany ? created : undefined,
        updated: updateMany ? updated : undefined,
        deleted: deleteMany ? deleted : undefined,
    };
}