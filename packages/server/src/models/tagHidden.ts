import { tagHiddensCreate, tagHiddensUpdate } from "@shared/validation";
import { relationshipBuilderHelper, RelationshipTypes } from "./builder";
import { TagModel } from "./tag";
import { TagHidden, TagHiddenCreateInput, TagHiddenUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

export const tagHiddenFormatter = (): FormatConverter<TagHidden, any> => ({
    relationshipMap: {
        __typename: 'TagHidden',
    },
})

export const tagHiddenMutater = (prisma: PrismaType) => ({
    async shapeBase(userId: string, data: TagHiddenCreateInput | TagHiddenUpdateInput) {
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
    async shapeRelationshipCreate(userId: string, data: TagHiddenCreateInput): Promise<Prisma.user_tag_hiddenCreateWithoutUserInput> {
        return {
            ...(await this.shapeBase(userId, data)),
        }
    },
    async shapeRelationshipUpdate(userId: string, data: TagHiddenUpdateInput): Promise<Prisma.user_tag_hiddenUpdateWithoutUserInput> {
        return {
            isBlur: data?.isBlur ?? undefined,
        }
    },
    async shapeCreate(userId: string, data: TagHiddenCreateInput): Promise<Prisma.user_tag_hiddenUpsertArgs['create']> {
        return {
            ...(await this.shapeBase(userId, data)),
            user: { connect: { id: userId } },
        }
    },
    async shapeUpdate(userId: string, data: TagHiddenUpdateInput): Promise<Prisma.user_tag_hiddenUpsertArgs['update']> {
        return {
            isBlur: data?.isBlur ?? undefined,
        }
    },
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'hiddenTags',
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd,
            isTransferable: false,
            shape: { shapeCreate: this.shapeRelationshipCreate, shapeUpdate: this.shapeRelationshipUpdate },
            userId,
        });
    },
    async cud(params: CUDInput<TagHiddenCreateInput, TagHiddenUpdateInput>): Promise<CUDResult<TagHidden>> {
        return cudHelper({
            ...params,
            objectType: 'TagHidden',
            prisma,
            yup: { yupCreate: tagHiddensCreate, yupUpdate: tagHiddensUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const TagHiddenModel = ({
    prismaObject: (prisma: PrismaType) => prisma.user_tag_hidden,
    format: tagHiddenFormatter(),
    mutate: tagHiddenMutater,
    type: 'TagHidden' as GraphQLModelType,
})