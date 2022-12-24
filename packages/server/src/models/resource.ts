import { ResourceSortBy } from "@shared/consts";
import { Resource, ResourceSearchInput, ResourceCreateInput, ResourceUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { ResourceListModel } from "./resourceList";
import { permissionsSelectHelper } from "../builders";
import { bestLabel } from "../utils";
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

// const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ResourceCreateInput | ResourceUpdateInput, isAdd: boolean) => {
//     return {
//         id: data.id,
//         index: data.index,
//         translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
//         usedFor: data.usedFor ?? undefined,
//     }
// }

// const mutater = (): Mutater<Model> => ({
//     shape: {
//         create: async ({ data, prisma, userData }) => {
//             return {
//                 ...await shapeBase(prisma, userData, data, true),
//                 link: data.link,
//                 listId: data.listId,
//             };
//         },
//         update: async ({ data, prisma, userData }) => {
//             return {
//                 ...await shapeBase(prisma, userData, data, false),
//                 link: data.link ?? undefined,
//                 listId: data.listId ?? undefined,
//             };
//         },
//     },
//     yup: resourceValidation,
// })

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const ResourceModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.resource,
    display: displayer(),
    format: {
        gqlRelMap: { __typename },
        prismaRelMap: { __typename },
        countFields: {},
    },
    mutate: {} as any,//mutater(),
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
        maxObjects: 50000,
        permissionsSelect: (...params) => ({
            id: true,
            ...permissionsSelectHelper([
                ['list', 'ResourceList'],
            ], ...params)
        }),
        permissionResolvers: (params) => ResourceListModel.validate!.permissionResolvers(params),
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