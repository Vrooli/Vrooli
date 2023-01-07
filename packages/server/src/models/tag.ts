import { tagValidation } from "@shared/validation";
import { TagSortBy } from "@shared/consts";
import { StarModel } from "./star";
import { Tag, TagSearchInput, TagCreateInput, TagUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";

// const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: TagCreateInput | TagUpdateInput, isAdd: boolean) => {
//     return {
//         tag: data.tag,
//         createdByUserId: userData.id,
//         translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
//     }
// }

// const mutater = (): Mutater<Model> => ({
//     shape: {
//         create: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, true),
//         update: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, false),
//     },
//     yup: tagValidation,
// })

const type = 'Tag' as const;
const suppFields = ['you.isOwn', 'you.isStarred'] as const;
export const TagModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: TagCreateInput,
    GqlUpdate: TagUpdateInput,
    GqlModel: Tag,
    GqlSearch: TagSearchInput,
    GqlSort: TagSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.tagUpsertArgs['create'],
    PrismaUpdate: Prisma.tagUpsertArgs['update'],
    PrismaModel: Prisma.tagGetPayload<SelectWrap<Prisma.tagSelect>>,
    PrismaSelect: Prisma.tagSelect,
    PrismaWhere: Prisma.tagWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.tag,
    display: {
        select: () => ({ id: true, tag: true }),
        label: (select) => select.tag,
    },
    format: {
        gqlRelMap: {
            type,
            apis: 'Api',
            notes: 'Note',
            organizations: 'Organization',
            posts: 'Post',
            projects: 'Project',
            reports: 'Report',
            routines: 'Routine',
            smartContracts: 'SmartContract',
            standards: 'Standard',
            starredBy: 'User',
        },
        prismaRelMap: {
            type,
            createdBy: 'User',
            apis: 'Api',
            notes: 'Note',
            organizations: 'Organization',
            posts: 'Post',
            projects: 'Project',
            reports: 'Report',
            routines: 'Routine',
            smartContracts: 'SmartContract',
            standards: 'Standard',
            starredBy: 'User',
            // scheduleFilters: 'ScheduleFilter',
        },
        joinMap: {
            apis: 'tagged',
            notes: 'tagged',
            organizations: 'tagged',
            posts: 'tagged',
            projects: 'tagged',
            reports: 'tagged',
            routines: 'tagged',
            smartContracts: 'tagged',
            standards: 'tagged',
            starredBy: 'user'
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ['createdByUserId', 'id'],
            toGraphQL: async ({ ids, objects, prisma, userData }) => ({
                'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, type),
                'you.isOwn': objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id),
            }),
        },
    },
    mutate: {} as any,//mutater(),
    search: {
        defaultSort: TagSortBy.StarsDesc,
        sortBy: TagSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            maxStars: true,
            minStars: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                'tagWrapped',
            ]
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: {
            User: 10000,
            Organization: 0,
        },
        permissionsSelect: () => ({ id: true }),
        permissionResolvers: () => ({}),
        owner: () => ({}),
        isDeleted: () => false,
        isPublic: () => true,
        profanityFields: ['tag'],
        visibility: {
            private: {},
            public: {},
            owner: () => ({}),
        },
    },
})