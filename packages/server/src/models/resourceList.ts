import { resourceListsCreate, resourceListsUpdate } from "@shared/validation";
import { ResourceListSortBy } from "@shared/consts";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Validator, Mutater, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { relBuilderHelper } from "../actions";
import { combineQueries, permissionsSelectHelper } from "../builders";
import { bestLabel, oneIsPublic, translationRelationshipBuilder } from "../utils";

const formatter = (): Formatter<ResourceList, any> => ({
    relationshipMap: {
        __typename: 'ResourceList',
        resources: 'Resource',
    },
})

const validator = (): Validator<
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
    isTransferable: false,
    maxObjects: 50000,
    permissionsSelect: (...params) => ({
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
        ], ...params)
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
    isPublic: (data, languages) => oneIsPublic<Prisma.resource_listSelect>(data, [
        // ['apiVersion', 'Api'],
        ['organization', 'Organization'],
        // ['post', 'Post'],
        ['projectVersion', 'Project'],
        ['routineVersion', 'Routine'],
        // ['smartContractVersion', 'SmartContract'],
        ['standardVersion', 'Standard'],
        // ['userSchedule', 'UserSchedule'],
    ], languages),
    visibility: {
        private: {},
        public: {},
        owner: (userId) => ({
            OR: [
                // { apiVersion: ApiModel.validate.visibility.owner(userId) },
                { organization: OrganizationModel.validate.visibility.owner(userId) },
                // { post: PostModel.validate.visibility.owner(userId) },
                { project: ProjectModel.validate.visibility.owner(userId) },
                { routineVersion: RoutineModel.validate.visibility.owner(userId) },
                // { smartContractVersion: SmartContractModel.validate.visibility.owner(userId) },
                { standardVersion: StandardModel.validate.visibility.owner(userId) },
                { userSchedule: { userId } },
            ]
        }),
    }
})

const searcher = (): Searcher<
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

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ResourceListCreateInput | ResourceListUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        organization: data.organizationId ? { connect: { id: data.organizationId } } : undefined,
        project: data.projectId ? { connect: { id: data.projectId } } : undefined,
        routine: data.routineId ? { connect: { id: data.routineId } } : undefined,
        user: data.userId ? { connect: { id: data.userId } } : undefined,
        resources: await relBuilderHelper({ data, isAdd, isOneToOne: false, isRequired: false, relationshipName: 'resources', objectType: 'Resource', prisma, userData }),
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}

const mutater = (): Mutater<
    ResourceList,
    { graphql: ResourceListCreateInput, db: Prisma.resource_listUpsertArgs['create'] },
    { graphql: ResourceListUpdateInput, db: Prisma.resource_listUpsertArgs['update'] },
    { graphql: ResourceListCreateInput, db: Prisma.resource_listCreateWithoutApiVersionInput },
    { graphql: ResourceListUpdateInput, db: Prisma.resource_listUpdateWithoutApiVersionInput }
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                ...await shapeBase(prisma, userData, data, true),
            };
        },
        update: async ({ data, prisma, userData }) => {
            return {
                ...await shapeBase(prisma, userData, data, false),
            };
        },
        relCreate: mutater().shape.create,
        relUpdate: mutater().shape.update,
    },
    yup: { create: resourceListsCreate, update: resourceListsUpdate },
})

const displayer = (): Displayer<
    Prisma.resource_listSelect,
    Prisma.resource_listGetPayload<{ select: { [K in keyof Required<Prisma.resource_listSelect>]: true } }>
> => ({
    select: { id: true, translations: { select: { language: true, title: true } } },
    label: (select, languages) => bestLabel(select.translations, 'title', languages),
})

export const ResourceListModel = ({
    delegate: (prisma: PrismaType) => prisma.resource_list,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'ResourceList' as GraphQLModelType,
    validate: validator(),
})