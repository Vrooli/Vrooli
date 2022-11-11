import { resourceListsCreate, resourceListsUpdate, resourceListTranslationsCreate, resourceListTranslationsUpdate } from "@shared/validation";
import { ResourceListSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, relationshipBuilderHelper, RelationshipTypes } from "./builder";
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
            organization: data.organizationId ? { connect: { id: data.organizationId } } : undefined,
            project: data.projectId ? { connect: { id: data.projectId } } : undefined,
            routine: data.routineId ? { connect: { id: data.routineId } } : undefined,
            user: data.userId ? { connect: { id: data.userId } } : undefined,
            resources: ResourceModel.mutate(prisma).relationshipBuilder(userId, data, isAdd),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: resourceListTranslationsCreate, update: resourceListTranslationsUpdate }, isAdd),
        };
    },
    shapeCreate(userId: string, data: ResourceListCreateInput): Prisma.resource_listUpsertArgs['create'] {
        return {
            ...this.shapeBase(userId, data, true),
        };
    },
    shapeUpdate(userId: string, data: ResourceListUpdateInput): Prisma.resource_listUpsertArgs['update'] {
        return {
            ...this.shapeBase(userId, data, false),
        };
    },
    relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resourceList',
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd,
            isOneToOne: true,
            // connect/disconnect not supported by resource lists (since they can only be applied to one object)
            relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect],
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            userId,
        });
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