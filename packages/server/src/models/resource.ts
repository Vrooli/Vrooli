import { resourceCreate, resourcesCreate, resourcesUpdate, resourceUpdate } from "@shared/validation";
import { ResourceSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, relationshipToPrisma, RelationshipTypes } from "./builder";
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
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resources',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape, excluding "createdFor" and "createdForId" fields
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resources (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Shape
        if (Array.isArray(formattedInput.create)) {
            // If title or description is not provided, try querying for the link's og tags TODO
            const creates: { [x: string]: any }[] = [];
            for (const create of formattedInput.create) {
                creates.push(this.shapeRelationshipCreate(userId, create as any));
            }
            formattedInput.create = creates;
        }
        if (Array.isArray(formattedInput.update)) {
            const updates: { where: { [x: string]: any }, data: { [x: string]: any } }[] = [];
            for (const update of formattedInput.update) {
                updates.push({
                    where: update.where,
                    data: this.shapeRelationshipUpdate(userId, update.data as any),
                })
            }
            formattedInput.update = updates;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async cud(params: CUDInput<ResourceCreateInput & { listId: string }, ResourceUpdateInput>): Promise<CUDResult<Resource>> {
        return cudHelper({
            ...params,
            objectType: 'Resource',
            prisma,
            prismaObject: (p) => p.resource as any,
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