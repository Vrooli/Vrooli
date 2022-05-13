import { CODE, tagHiddensCreate, tagHiddensUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { genErrorCode } from "../../logger";
import { PrismaType } from "types";
import { Count, TagHidden, TagHiddenCreateInput, TagHiddenUpdateInput } from "../../schema/types";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper, ValidateMutationsInput } from "./base";
import { TagModel } from "./tag";

//==============================================================
/* #region Custom Components */
//==============================================================

export const tagHiddenFormatter = (): FormatConverter<TagHidden> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.TagHidden,
        'user': GraphQLModelType.User,
    },
})

export const tagHiddenMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string, data: TagHiddenCreateInput): Promise<any> {
        return {
            userId,
            isBlur: data.isBlur ?? false,
            tags: await TagModel(prisma).relationshipBuilder(userId, data, GraphQLModelType.TagHidden),
        }
    },
    async toDBShapeUpdate(userId: string, data: TagHiddenUpdateInput): Promise<any> {
        return {
            isBlur: data?.isBlur ?? undefined,
        }
    },
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'hiddenTags',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by TagHiddens (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] });
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        // Validate create
        if (Array.isArray(createMany)) {
            tagHiddensCreate.validateSync(createMany, { abortEarly: false });
            // Check for max tags TODO
            for (let data of createMany) {
                // Convert nested relationships
                data = {
                    ...data,
                    tags: await TagModel(prisma).relationshipBuilder(userId, data, GraphQLModelType.TagHidden),
                }
            }
        }
        // Validate update
        if (Array.isArray(updateMany)) {
            tagHiddensUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            for (let data of updateMany) {
                // Convert nested relationships
                data.data = {
                    ...data.data,
                    tags: await TagModel(prisma).relationshipBuilder(userId, data.data, GraphQLModelType.TagHidden),
                }
            }
        }
        return Object.keys(formattedInput).length > 0 ? {
            create: createMany,
            update: updateMany,
            delete: deleteMany
        } : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<TagHiddenCreateInput, TagHiddenUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0187') });
        if (createMany) {
            tagHiddensCreate.validateSync(createMany, { abortEarly: false });
            // Check for max tags TODO
        }
        if (updateMany) {
            tagHiddensUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
        }
    },
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<TagHiddenCreateInput, TagHiddenUpdateInput>): Promise<CUDResult<TagHidden>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShapeAdd(userId ?? '', input);
                // Create object
                const currCreated = await prisma.user_tag_hidden.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.user_tag_hidden.findFirst({
                    where: { ...input.where, userId: userId ?? '' },
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown, 'Could not find TagHidden to update', { code: genErrorCode('0188') });
                // Update object
                const currUpdated = await prisma.user_tag_hidden.update({
                    where: input.where,
                    data: await this.toDBShapeUpdate(userId ?? '', input.data),
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            const tagHiddens = await prisma.user_tag_hidden.findMany({
                where: { id: { in: deleteMany } },
            })
            if (tagHiddens.some(t => t.userId !== userId)) throw new CustomError(CODE.Unauthorized, "You can't delete TagHiddens that don't belong to you", { code: genErrorCode('0189') });
            deleted = await prisma.user_tag_hidden.deleteMany({
                where: { id: { in: tagHiddens.map(t => t.id) } },
            });
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TagHiddenModel(prisma: PrismaType) {
    const prismaObject = prisma.wallet;
    const format = tagHiddenFormatter();
    const mutate = tagHiddenMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...mutate
    }
}

//==============================================================
/* #endregion Model */
//==============================================================