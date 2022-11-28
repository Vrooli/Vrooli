import { resourcesCreate, resourcesUpdate } from "@shared/validation";
import { ResourceSortBy } from "@shared/consts";
import { Resource, ResourceSearchInput, ResourceCreateInput, ResourceUpdateInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Mutater, Validator, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { ResourceListModel } from "./resourceList";
import { combineQueries, permissionsSelectHelper } from "../builders";
import { bestLabel, translationRelationshipBuilder } from "../utils";

const formatter = (): Formatter<Resource, any> => ({
    relationshipMap: { __typename: 'Resource' }, // For now, resource is never queried directly. So no need to handle relationships
})

const searcher = (): Searcher<
    ResourceSearchInput,
    ResourceSortBy,
    Prisma.resourceOrderByWithRelationInput,
    Prisma.resourceWhereInput
> => ({
    defaultSort: ResourceSortBy.IndexAsc,
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
            { link: { ...insensitive } },
        ]
    }),
    customQueries(input) {
        // const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        ])
    },
})

const validator = (): Validator<
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource,
    Prisma.resourceGetPayload<{ select: { [K in keyof Required<Prisma.resourceSelect>]: true } }>,
    any,
    Prisma.resourceSelect,
    Prisma.resourceWhereInput
> => ({
    validateMap: {
        __typename: 'Resource',
        list: 'ResourceList',
    },
    isTransferable: false,
    maxObjects: 50000,
    permissionsSelect: (...params) => ({
        id: true,
        ...permissionsSelectHelper([
            ['list', 'ResourceList'],
        ], ...params)
    }),
    permissionResolvers: (params) => ResourceListModel.validate.permissionResolvers(params),
    owner: (data) => ResourceListModel.validate.owner(data.list as any),
    isDeleted: () => false,
    isPublic: (data, languages) => ResourceListModel.validate.isPublic(data.list as any, languages),
    visibility: {
        private: {},
        public: {},
        owner: (userId) => ({ list: ResourceListModel.validate.visibility.owner(userId) }),
    }
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ResourceCreateInput | ResourceUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        index: data.index,
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
        usedFor: data.usedFor ?? undefined,
    }
}

const mutater = (): Mutater<
    Resource,
    { graphql: ResourceCreateInput, db: Prisma.resourceUpsertArgs['create'] },
    { graphql: ResourceUpdateInput, db: Prisma.resourceUpsertArgs['update'] },
    { graphql: ResourceCreateInput, db: Prisma.resourceCreateWithoutListInput },
    { graphql: ResourceUpdateInput, db: Prisma.resourceUpdateWithoutListInput }
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                ...await shapeBase(prisma, userData, data, true),
                link: data.link,
                listId: data.listId,
            };
        },
        update: async ({ data, prisma, userData }) => {
            return {
                ...await shapeBase(prisma, userData, data, false),
                link: data.link ?? undefined,
                listId: data.listId ?? undefined,
            };
        },
        relCreate: async ({ data, prisma, userData }) => {
            return {
                ...await shapeBase(prisma, userData, data, true),
                link: data.link,
            }
        },
        relUpdate: async ({ data, prisma, userData }) => {
            return {
                ...await shapeBase(prisma, userData, data, false),
                link: data.link ?? undefined,
            }
        }
    },
    yup: { create: resourcesCreate, update: resourcesUpdate },
})

const displayer = (): Displayer<
    Prisma.resourceSelect,
    Prisma.resourceGetPayload<{ select: { [K in keyof Required<Prisma.resourceSelect>]: true } }>
> => ({
    select: { id: true, translations: { select: { language: true, title: true } } },
    label: (select, languages) => bestLabel(select.translations, 'title', languages),
})

export const ResourceModel = ({
    delegate: (prisma: PrismaType) => prisma.resource,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'Resource' as GraphQLModelType,
    validate: validator(),
})