import { resourceCreate, resourcesCreate, resourcesUpdate, resourceUpdate } from "@shared/validation";
import { ResourceSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, relationshipBuilderHelper, RelationshipTypes } from "./builder";
import { TranslationModel } from "./translation";
import { organizationQuerier } from "./organization";
import { Resource, ResourceSearchInput, ResourceCreateInput, ResourceUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

export const resourceFormatter = (): FormatConverter<Resource, any> => ({
    relationshipMap: { '__typename': 'Resource' }, // For now, resource is never queried directly. So no need to handle relationships
})

export const resourceSearcher = (): Searcher<ResourceSearchInput> => ({
    defaultSort: ResourceSortBy.IndexAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ResourceSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ResourceSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ResourceSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ResourceSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ResourceSortBy.IndexAsc]: { index: 'asc' },
            [ResourceSortBy.IndexDesc]: { index: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                    { link: { ...insensitive } },
                ]
            })
        })
    },
    customQueries(input: ResourceSearchInput): { [x: string]: any } {
        // const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        ])
    },
})

// TODO create proper permissioner
export const resourcePermissioner = () => ({
    ownershipQuery: (userId: string) => ({
        OR: [
            organizationQuerier().hasRoleInOrganizationQuery(userId),
            { user: { id: userId } }
        ]
    })
})

export const resourceMutater = (prisma: PrismaType) => ({
    shapeBase(userId: string, data: ResourceCreateInput | ResourceUpdateInput, isAdd: boolean) {
        return {
            id: data.id,
            index: data.index,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: resourceCreate, update: resourceUpdate }, isAdd),
        };
    },
    shapeRelationshipCreate(userId: string, data: ResourceCreateInput): Prisma.resourceCreateWithoutListInput {
        return {
            ...this.shapeBase(userId, data, true),
            link: data.link,
            usedFor: data.usedFor,
        };
    },
    shapeRelationshipUpdate(userId: string, data: ResourceUpdateInput): Prisma.resourceUpdateWithoutListInput {
        return {
            ...this.shapeBase(userId, data, false),
            link: data.link ?? undefined,
            usedFor: data.usedFor ?? undefined,
        };
    },
    shapeCreate(userId: string, data: ResourceCreateInput & { listId: string }): Prisma.resourceUpsertArgs['create'] {
        return {
            ...this.shapeRelationshipCreate(userId, data),
            listId: data.listId,
        };
    },
    shapeUpdate(userId: string, data: ResourceUpdateInput): Prisma.resourceUpsertArgs['update'] {
        return {
            ...this.shapeRelationshipUpdate(userId, data),
        };
    },
    relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resources',
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
    async cud(params: CUDInput<ResourceCreateInput & { listId: string }, ResourceUpdateInput>): Promise<CUDResult<Resource>> {
        return cudHelper({
            ...params,
            objectType: 'Resource',
            prisma,
            yup: { yupCreate: resourcesCreate, yupUpdate: resourcesUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const ResourceModel = ({
    prismaObject: (prisma: PrismaType) => prisma.resource,
    format: resourceFormatter(),
    mutate: resourceMutater,
    search: resourceSearcher(),
    type: 'Resource' as GraphQLModelType,
})