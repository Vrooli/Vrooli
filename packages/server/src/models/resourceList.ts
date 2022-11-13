import { resourceListsCreate, resourceListsUpdate, resourceListTranslationsCreate, resourceListTranslationsUpdate } from "@shared/validation";
import { ResourceListSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, relationshipBuilderHelper } from "./builder";
import { TranslationModel } from "./translation";
import { ResourceModel } from "./resource";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";
import { organizationValidator } from "./organization";
import { projectValidator } from "./project";
import { routineValidator } from "./routine";
import { standardValidator } from "./standard";

export const resourceListFormatter = (): FormatConverter<ResourceList, any> => ({
    relationshipMap: {
        '__typename': 'ResourceList',
        'resources': 'Resource',
    },
})

export const resourceListValidator = (): Validator<ResourceListCreateInput, ResourceListUpdateInput, ResourceList, Prisma.resource_listSelect, Prisma.resource_listWhereInput> => ({
    validatedRelationshipMap: {
        // 'api': 'Api',
        'organization': 'Organization',
        // 'post': 'Post',
        'project': 'Project',
        'routine': 'Routine',
        // 'smartContract': 'SmartContract',
        // 'standard': 'Standard',
        // 'userSchedule': 'UserSchedule',
    },
    permissionsSelect: {
        id: true,
        // apiVersion: { select: apiValidator().permissionsSelect },
        organization: { select: organizationValidator().permissionsSelect },
        // post: { select: postValidator().permissionsSelect },
        project: { select: projectValidator().permissionsSelect },
        routineVersion: { select: routineValidator().permissionsSelect },
        // smartContractVersion: { select: smartContractValidator().permissionsSelect },
        standardVersion: { select: standardValidator().permissionsSelect },
        // userScheduledRoutine: { select: routineValidator().permissionsSelect },
    },
    ownerOrMemberWhere: (userId) => ({
        OR: [
            // { apiVersion: apiValidator().ownerOrMemberWhere(userId) },
            { organization: organizationValidator().ownerOrMemberWhere(userId) },
            // { post: postValidator().ownerOrMemberWhere(userId) },
            { project: projectValidator().ownerOrMemberWhere(userId) },
            { routineVersion: routineValidator().ownerOrMemberWhere(userId) },
            // { smartContractVersion: smartContractValidator().ownerOrMemberWhere(userId) },
            { standardVersion: standardValidator().ownerOrMemberWhere(userId) },
            { userSchedule: { userId } },
        ]
    }),
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
            isTransferable: false,
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            userId,
        });
    },
    async cud(params: CUDInput<ResourceListCreateInput, ResourceListUpdateInput>): Promise<CUDResult<ResourceList>> {
        return cudHelper({
            ...params,
            objectType: 'ResourceList',
            prisma,
            yup: { yupCreate: resourceListsCreate, yupUpdate: resourceListsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const ResourceListModel = ({
    prismaObject: (prisma: PrismaType) => prisma.resource_list,
    format: resourceListFormatter(),
    mutate: resourceListMutater,
    search: resourceListSearcher(),
    type: 'ResourceList' as GraphQLModelType,
    validate: resourceListValidator(),
})