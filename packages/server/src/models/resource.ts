import { MaxObjects, ResourceSortBy } from "@shared/consts";
import { Resource, ResourceSearchInput, ResourceCreateInput, ResourceUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { ResourceListModel } from "./resourceList";
import { bestLabel, translationShapeHelper } from "../utils";
import { SelectWrap } from "../builders/types";
import { resourceValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ResourceCreateInput,
    GqlUpdate: ResourceUpdateInput,
    GqlModel: Resource,
    GqlSearch: ResourceSearchInput,
    GqlSort: ResourceSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.resourceUpsertArgs['create'],
    PrismaUpdate: Prisma.resourceUpsertArgs['update'],
    PrismaModel: Prisma.resourceGetPayload<SelectWrap<Prisma.resourceSelect>>,
    PrismaSelect: Prisma.resourceSelect,
    PrismaWhere: Prisma.resourceWhereInput,
}

const __typename = 'Resource' as const;
const suppFields = [] as const;
export const ResourceModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.resource,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: { __typename },
        prismaRelMap: { __typename },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                link: data.link,
                usedFor: data.usedFor,
                ...(await shapeHelper({ relation: 'list', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'ResourceList', parentRelationshipName: 'resources', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                link: noNull(data.link),
                usedFor: noNull(data.usedFor),
                ...(await shapeHelper({ relation: 'list', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'resources', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            }),
        },
        yup: resourceValidation,
    },
    search: {
        defaultSort: ResourceSortBy.IndexAsc,
        sortBy: ResourceSortBy,
        searchFields: {
            createdTimeFrame: true,
            resourceListId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                'transNameWrapped',
                'linkWrapped',
            ]
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            list: 'ResourceList',
        }),
        permissionResolvers: (params) => ResourceListModel.validate!.permissionResolvers({ ...params, data: params.data.list as any }),
        owner: (data) => ResourceListModel.validate!.owner(data.list as any),
        isDeleted: () => false,
        isPublic: (data, languages) => ResourceListModel.validate!.isPublic(data.list as any, languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ list: ResourceListModel.validate!.visibility.owner(userId) }),
        }
    },
})