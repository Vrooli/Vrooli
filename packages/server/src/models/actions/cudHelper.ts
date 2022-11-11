import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { Count } from "../../schema/types";
import { PrismaDelegate } from "../../types";
import { modelToGraphQL, ObjectMap, selectHelper } from "../builder";
import { CUDHelperInput, CUDResult, Validator } from "../types";
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
    userId,
    yup,
}: CUDHelperInput<GQLCreate, GQLUpdate, GQLObject, DBCreate, DBUpdate>): Promise<CUDResult<GQLObject>> {
    // Perform mutations
    let created: GQLObject[] = [], updated: GQLObject[] = [], deleted: Count = { count: 0 };
    // Perform validations
    const validator: Validator<GQLObject, any> | undefined = ObjectMap[objectType].validator;
    // Validate yup
    createMany && yup.yupCreate.validateSync(createMany, { abortEarly: false });
    updateMany && yup.yupUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
    // Profanity check
    createMany && profanityCheck(createMany, partialInfo);
    updateMany && profanityCheck(updateMany.map(u => u.data), partialInfo);
    // Max objects check
    await maxObjectsCheck({ userId, createMany: createMany as any, updateMany: updateMany as any, deleteMany, prisma, objectType, maxCount: 1000000 });
    // Shape and perform cud
    const prismaDelegate: PrismaDelegate = ObjectMap[objectType].prismaObject(prisma);
    if (createMany) {
        // Permissions check
        const isPermitted = await permissionsCheck({ actions: ['Create'], objectIds: createMany.map(c => c.id), objectType, prisma, userId });
        if (!isPermitted) throw new CustomError(CODE.Unauthorized, `Not allowed to create object`, { code: genErrorCode('0276') });
        // Perform custom validation
        validator?.validations?.create && await validator.validations.create(createMany, userId);
        for (const create of createMany) {
            // Shape 
            const data = await shape.shapeCreate(userId, create);
            // Create
            const select = await prismaDelegate.create({
                data: data as any,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL<GQLObject>(select, partialInfo);
            created = created ? [...created, converted] : [converted];
        }
        // Call onCreated
        onCreated && await onCreated(created);
    }
    if (updateMany) {
        // Permissions check
        const isPermitted = await permissionsCheck({ actions: ['Update'], objectIds: updateMany.map(u => u.data.id), objectType, prisma, userId });
        if (!isPermitted) throw new CustomError(CODE.Unauthorized, `Not allowed to update object`, { code: genErrorCode('0277') });
        // Perform custom validation
        validator?.validations?.update && await validator.validations.update(updateMany, userId);
        for (const update of updateMany) {
            // Shape
            const data = await shape.shapeUpdate(userId, update.data);
            // Update
            const select = await prismaDelegate.update({
                where: update.where,
                data: data as any,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL<GQLObject>(select, partialInfo);
            updated = updated ? [...updated, converted] : [converted];
        }
        // Call onUpdated
        onUpdated && await onUpdated(updated, updateMany.map(u => u.data));
    }
    if (deleteMany) {
        // Permissions check
        const isPermitted = await permissionsCheck({ actions: ['Delete'], objectIds: deleteMany, objectType, prisma, userId });
        if (!isPermitted) throw new CustomError(CODE.Unauthorized, `Not allowed to delete object`, { code: genErrorCode('0278') });
        // Perform custom validation
        validator?.validations?.delete && await validator.validations.delete(deleteMany, userId);
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