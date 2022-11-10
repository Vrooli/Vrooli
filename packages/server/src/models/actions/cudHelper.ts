import { Count } from "../../schema/types";
import { modelToGraphQL, selectHelper, validateMaxObjects, validateObjectOwnership } from "../builder";
import { TranslationModel } from "../translation";
import { CUDHelperInput, CUDResult } from "../types";

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
export async function cudHelper<GQLCreate extends { [x: string]: any }, GQLUpdate extends { [x: string]: any }, GQLObject, DBCreate extends { [x: string]: any }, DBUpdate extends { [x: string]: any }>({
    objectType,
    partialInfo,
    prisma,
    prismaObject,
    userId,
    createMany,
    updateMany,
    deleteMany,
    yup,
    shape,
}: CUDHelperInput<GQLCreate, GQLUpdate, GQLObject, DBCreate, DBUpdate>): Promise<CUDResult<GQLObject>> {
    // Perform mutations
    let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
    // Validate yup
    createMany && yup.yupCreate.validateSync(createMany, { abortEarly: false });
    updateMany && yup.yupUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
    // Profanity check
    createMany && TranslationModel.profanityCheck(createMany);
    updateMany && TranslationModel.profanityCheck(updateMany.map(u => u.data));
    // Permissions check
    await validateObjectOwnership({ userId, createMany: createMany as any, updateMany: updateMany as any, deleteMany, prisma, objectType });
    // Max objects check
    await validateMaxObjects({ userId, createMany: createMany as any, updateMany: updateMany as any, deleteMany, prisma, objectType, maxCount: 1000000 });
    // Shape and perform cud
    if (createMany) {
        for (const create of createMany) {
            // Shape 
            const data = await shape.shapeCreate(userId, create);
            // Create
            const select = await prismaObject(prisma).create({
                data: data as any,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL(select, partialInfo);
            created = created ? [...created, converted] : [converted];
        }
    }
    if (updateMany) {
        for (const update of updateMany) {
            // Shape
            const data = await shape.shapeUpdate(userId, update.data);
            // Update
            const select = await prismaObject(prisma).update({
                where: update.where,
                data: data as any,
                ...selectHelper(partialInfo)
            });
            // Convert
            const converted = modelToGraphQL(select, partialInfo);
            updated = updated ? [...updated, converted] : [converted];
        }
    }
    if (deleteMany) {
        deleted = await prismaObject(prisma).deleteMany({
            where: { id: { in: deleteMany } }
        })
    }
    return {
        created: createMany ? created : undefined,
        updated: updateMany ? updated : undefined,
        deleted: deleteMany ? deleted : undefined,
    };
}