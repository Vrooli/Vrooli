import { resourceListsCreate, resourceListsUpdate, resourceListTranslationsCreate, resourceListTranslationsUpdate } from "@shared/validation";
import { ResourceListSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, relationshipToPrisma, RelationshipTypes } from "./builder";
import { TranslationModel } from "./translation";
import { ResourceModel } from "./resource";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Permissioner } from "./types";
import { Prisma } from "@prisma/client";
import { routinePermissioner } from "./routine";
import { organizationPermissioner } from "./organization";
import { projectPermissioner } from "./project";
import { standardPermissioner } from "./standard";
import { cudHelper } from "./actions";

export const resourceListFormatter = (): FormatConverter<ResourceList, any> => ({
    relationshipMap: {
        '__typename': 'ResourceList',
        'resources': 'Resource',
    },
})

export const resourceListPermissioner = (): Permissioner<any, ResourceListSearchInput> => ({
    async get() {
        return [] as any;
    },
    async canSearch() {
        //TODO
        return 'full';
    },
    ownershipQuery: (userId) => ({
        OR: [
            // { apiVersion: apiPermissioner().ownershipQuery(userId) },
            { organization: organizationPermissioner().ownershipQuery(userId) },
            // { post: postPermissioner().ownershipQuery(userId) },
            { project: projectPermissioner().ownershipQuery(userId) },
            { routineVersion: routinePermissioner().ownershipQuery(userId) },
            // { smartContractVersion: smartContractPermissioner().ownershipQuery(userId) },
            { standardVersion: standardPermissioner().ownershipQuery(userId) },
            { userSchedule: { userId } },
        ]
    })
})

export const resourceListSearcher = (): Searcher<ResourceListSearchInput> => ({
    defaultSort: ResourceListSortBy.IndexAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ResourceListSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ResourceListSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ResourceListSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ResourceListSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ResourceListSortBy.IndexAsc]: { index: 'asc' },
            [ResourceListSortBy.IndexDesc]: { index: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                ]
            })
        })
    },
    customQueries(input: ResourceListSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        ])
    },
})

export const resourceListMutater = (prisma: PrismaType) => ({
    shapeBase(userId: string, data: ResourceListCreateInput | ResourceListUpdateInput, isAdd: boolean) {
        return {
            id: data.id,
            organizationId: data.organizationId ?? undefined,
            projectId: data.projectId ?? undefined,
            routineId: data.routineId ?? undefined,
            userId: data.userId ?? undefined,
            resources: ResourceModel.mutate(prisma).relationshipBuilder(userId, data, isAdd),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: resourceListTranslationsCreate, update: resourceListTranslationsUpdate }, isAdd),
        };
    },
    shapeCreate(userId: string, data: ResourceListCreateInput, isAdd: boolean): Prisma.resource_listUpsertArgs['create'] {
        return {
            ...this.shapeBase(userId, data, isAdd),
        };
    },
    shapeUpdate(userId: string, data: ResourceListUpdateInput, isAdd: boolean): Prisma.resource_listUpsertArgs['update'] {
        return {
            ...this.shapeBase(userId, data, isAdd),
        };
    },
    relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resourceList',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape. Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resource lists (since they can only be applied to one object)
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Shape
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            const create = formattedInput.create[0];
            // If title or description is not provided, try querying for the link's og tags TODO
            formattedInput.create = this.shapeCreate(userId, create, true);
        }
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            const update = formattedInput.update[0].data;
            formattedInput.update = {
                where: update.where,
                data: this.shapeUpdate(userId, update.data, false),
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async cud(params: CUDInput<ResourceListCreateInput, ResourceListUpdateInput>): Promise<CUDResult<ResourceList>> {
        return cudHelper({
            ...params,
            objectType: 'ResourceList',
            prisma,
            prismaObject: (p) => p.resource_list,
            yup: { yupCreate: resourceListsCreate, yupUpdate: resourceListsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const ResourceListModel = ({
    prismaObject: (prisma: PrismaType) => prisma.resource_list,
    format: resourceListFormatter(),
    mutate: resourceListMutater,
    permissions: resourceListPermissioner,
    search: resourceListSearcher(),
    type: 'ResourceList' as GraphQLModelType,
})