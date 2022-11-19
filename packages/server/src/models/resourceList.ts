import { resourceListsCreate, resourceListsUpdate, resourceListTranslationsCreate, resourceListTranslationsUpdate } from "@shared/validation";
import { ResourceListSortBy } from "@shared/consts";
import { combineQueries, permissionsSelectHelper, relationshipBuilderHelper } from "./builder";
import { TranslationModel } from "./translation";
import { ResourceModel } from "./resource";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";
import { organizationValidator } from "./organization";
import { projectValidator } from "./project";
import { routineValidator } from "./routine";
import { standardValidator } from "./standard";
import { oneIsPublic } from "./utils";

export const resourceListFormatter = (): FormatConverter<ResourceList, any> => ({
    relationshipMap: {
        __typename: 'ResourceList',
        resources: 'Resource',
    },
})

export const resourceListValidator = (): Validator<
    ResourceListCreateInput,
    ResourceListUpdateInput,
    ResourceList,
    Prisma.resource_listGetPayload<{ select: { [K in keyof Required<Prisma.resource_listSelect>]: true } }>,
    any,
    Prisma.resource_listSelect,
    Prisma.resource_listWhereInput
> => ({
    validateMap: {
        __typename: 'ResourceList',
        // api: 'Api',
        organization: 'Organization',
        // post: 'Post',
        projectVersion: 'Project',
        routineVersion: 'Routine',
        // smartContract: 'SmartContract',
        // standard: 'Standard',
        // userSchedule: 'UserSchedule',
    },
    permissionsSelect: (userId) => ({
        id: true,
        ...permissionsSelectHelper([
            // ['apiVersion', 'Api'],
            ['organization', 'Organization'],
            // ['post', 'Post'],
            ['projectVersion', 'Project'],
            ['routineVersion', 'Routine'],
            // ['smartContractVersion', 'SmartContract'],
            ['standardVersion', 'Standard'],
            // ['userSchedule', 'UserSchedule'],
        ], userId)
    }),
    permissionResolvers: ({ isAdmin }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
    ]),
    owner: (data) => ({
        Organization: data.organization,
        User: (data.userSchedule as any)?.user,
    }),
    isDeleted: () => false,
    isPublic: (data) => oneIsPublic<Prisma.resource_listSelect>(data, [
        // ['apiVersion', 'Api'],
        ['organization', 'Organization'],
        // ['post', 'Post'],
        ['projectVersion', 'Project'],
        ['routineVersion', 'Routine'],
        // ['smartContractVersion', 'SmartContract'],
        ['standardVersion', 'Standard'],
        // ['userSchedule', 'UserSchedule'],
    ]),
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

export const resourceListSearcher = (): Searcher<
    ResourceListSearchInput,
    ResourceListSortBy,
    Prisma.resource_listOrderByWithRelationInput,
    Prisma.resource_listWhereInput
> => ({
    defaultSort: ResourceListSortBy.IndexAsc,
    sortMap: {
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        IndexAsc: { index: 'asc' },
        IndexDesc: { index: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
            { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
        ]
    }),
    customQueries(input) {
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        ])
    },
})

export const resourceListMutater = (prisma: PrismaType): Mutater<ResourceList> => ({
    shapeBase(userId: string, data: ResourceListCreateInput | ResourceListUpdateInput, isAdd: boolean) {
        return {
            id: data.id,
            organization: data.organizationId ? { connect: { id: data.organizationId } } : undefined,
            project: data.projectId ? { connect: { id: data.projectId } } : undefined,
            routine: data.routineId ? { connect: { id: data.routineId } } : undefined,
            user: data.userId ? { connect: { id: data.userId } } : undefined,
            resources: ResourceModel.mutate(prisma).relationshipBuilder!(userId, data, isAdd),
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