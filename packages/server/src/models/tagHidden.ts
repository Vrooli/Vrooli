import { tagHiddensCreate, tagHiddensUpdate } from "@shared/validation";
import { CODE } from "@shared/consts";
import { modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper } from "./builder";
import { TagModel } from "./tag";
import { CustomError, genErrorCode } from "../events";
import { TagHidden, TagHiddenCreateInput, TagHiddenUpdateInput, Count } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";

export const tagHiddenFormatter = (): FormatConverter<TagHidden, any> => ({
    relationshipMap: {
        '__typename': 'TagHidden',
    },
})

export const tagHiddenMutater = (prisma: PrismaType) => ({
    async toDBBase(userId: string, data: TagHiddenCreateInput | TagHiddenUpdateInput) {
        // Tags are built as many-to-many, but in this case we want a one-to-one relationship. 
        // So we must modify the data a bit.
        const tagData = await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'TagHidden', 'tag');
        let tag: any = tagData && Array.isArray(tagData.create) ? tagData.create[0].tag : undefined;
        return {
            id: data.id,
            isBlur: data.isBlur ?? false,
            tag,
        }
    },
    async toDBRelationshipCreate(userId: string, data: TagHiddenCreateInput): Promise<Prisma.user_tag_hiddenCreateWithoutUserInput> {
        return {
            ...(await this.toDBBase(userId, data)),
        }
    },
    async toDBRelationshipUpdate(userId: string, data: TagHiddenUpdateInput): Promise<Prisma.user_tag_hiddenUpdateWithoutUserInput> {
        return {
            isBlur: data?.isBlur ?? undefined,
        }
    },
    async toDBCreate(userId: string, data: TagHiddenCreateInput): Promise<Prisma.user_tag_hiddenUpsertArgs['create']> {
        return {
            ...(await this.toDBBase(userId, data)),
            user: { connect: { id: userId } },
        }
    },
    async toDBUpdate(userId: string, data: TagHiddenUpdateInput): Promise<Prisma.user_tag_hiddenUpsertArgs['update']> {
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
            let result: any[] = [];
            for (let data of createMany) {
                // Convert nested relationships
                result.push(await this.toDBRelationshipCreate(userId, data as any))
            }
            createMany = result;
        }
        // Validate update
        if (Array.isArray(updateMany)) {
            tagHiddensUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            let result: any[] = [];
            for (let data of updateMany) {
                // Convert nested relationships
                result.push(await this.toDBRelationshipUpdate(userId, data.data as TagHiddenUpdateInput))
            }
            updateMany = updateMany.map(u => ({
                where: u.where,
                data: result.shift(),
            }))
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
        if (createMany) {
            tagHiddensCreate.validateSync(createMany, { abortEarly: false });
            // Check for max tags TODO
        }
        if (updateMany) {
            tagHiddensUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<TagHiddenCreateInput, TagHiddenUpdateInput>): Promise<CUDResult<TagHidden>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBCreate(userId, input);
                // Create object
                const currCreated = await prisma.user_tag_hidden.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.user_tag_hidden.findFirst({
                    where: { ...input.where, userId: userId },
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown, 'Could not find TagHidden to update', { code: genErrorCode('0188') });
                // Update object
                const currUpdated = await prisma.user_tag_hidden.update({
                    where: input.where,
                    data: await this.toDBUpdate(userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
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

export const TagHiddenModel = ({
    prismaObject: (prisma: PrismaType) => prisma.wallet,
    format: tagHiddenFormatter(),
    mutate: tagHiddenMutater,
    type: 'TagHidden' as GraphQLModelType,
})