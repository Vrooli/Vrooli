import { tagsCreate, tagsUpdate } from "@shared/validation";
import { TagSortBy } from "@shared/consts";
import { StarModel } from "./star";
import { Tag, TagSearchInput, TagCreateInput, TagUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Mutater, Validator, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";

type Model = {
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
}

const __typename = 'Tag' as const;

const suppFields = ['isOwn', 'isStarred'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
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
        __typename,
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
    supplemental: {
        graphqlFields: suppFields,
        dbFields: ['createdByUserId', 'id'],
        toGraphQL: ({ ids, objects, prisma, userData }) => ({
            isStarred: async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
            isOwn: async () => objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id),
        }),
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: TagSortBy.StarsDesc,
    sortBy: TagSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'excludeIds',
        'minStars',
        'translationLanguages',
        'updatedTimeFrame',
    ],
    searchStringQuery: () => ({
        OR: [
            'transDescriptionWrapped',
            'tagWrapped',
        ]
    }),
})

const validator = (): Validator<Model> => ({
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
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: TagCreateInput | TagUpdateInput, isAdd: boolean) => {
    return {
        tag: data.tag,
        createdByUserId: userData.id,
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, true),
        update: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, false),
    },
    yup: { create: tagsCreate, update: tagsUpdate },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, tag: true }),
    label: (select) => select.tag,
})

export const TagModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.tag,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    validate: validator(),
})