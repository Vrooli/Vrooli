import { resourcesCreate, resourcesUpdate } from "@shared/validation";
import { ResourceSortBy } from "@shared/consts";
import { Resource, ResourceSearchInput, ResourceCreateInput, ResourceUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Mutater, Validator, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { ResourceListModel } from "./resourceList";
import { permissionsSelectHelper } from "../builders";
import { bestLabel, translationRelationshipBuilder } from "../utils";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ResourceCreateInput,
    GqlUpdate: ResourceUpdateInput,
    GqlModel: Resource,
    GqlSearch: ResourceSearchInput,
    GqlSort: ResourceSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.resourceUpsertArgs['create'],
    PrismaUpdate: Prisma.resourceUpsertArgs['update'],
    PrismaModel: Prisma.resourceGetPayload<SelectWrap<Prisma.resourceSelect>>,
    PrismaSelect: Prisma.resourceSelect,
    PrismaWhere: Prisma.resourceWhereInput,
}

const __typename = 'Resource' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: { __typename }, // For now, resource is never queried directly. So no need to handle relationships
})

const searcher = (): Searcher<Model> => ({
    defaultSort: ResourceSortBy.IndexAsc,
    sortBy: ResourceSortBy,
    searchFields: [
        'createdTimeFrame',
        'resourceListId',
        'translationLanguages',
        'updatedTimeFrame',
    ],
    
    searchStringQuery: () => ({
        OR: [
            'transDescriptionWrapped',
            'transNameWrapped',
            'linkWrapped',
        ]
    }),
})

const validator = (): Validator<Model> => ({
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

const mutater = (): Mutater<Model> => ({
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
    },
    yup: { create: resourcesCreate, update: resourcesUpdate },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const ResourceModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.resource,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    validate: validator(),
})