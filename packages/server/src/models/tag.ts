import { tagsCreate, tagsUpdate } from "@shared/validation";
import { TagSortBy } from "@shared/consts";
import { StarModel } from "./star";
import { Tag, TagSearchInput, TagCreateInput, TagUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Mutater, Validator, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { translationRelationshipBuilder } from "../utils";
import { SelectWrap } from "../builders/types";

const __typename = 'Tag' as const;

const suppFields = ['isOwn', 'isStarred'] as const;
const formatter = (): Formatter<Tag, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        starredBy: 'User',
    },
    joinMap: { organizations: 'tagged', projects: 'tagged', routines: 'tagged', standards: 'tagged', starredBy: 'user' },
    supplemental: {
        graphqlFields: suppFields,
        dbFields: ['createdByUserId', 'id'],
        toGraphQL: ({ ids, objects, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isOwn', async () => objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id)],
        ],
    },
})

const searcher = (): Searcher<
    TagSearchInput,
    TagSortBy,
    Prisma.tagWhereInput
> => ({
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

const validator = (): Validator<
    TagCreateInput,
    TagUpdateInput,
    Prisma.tagGetPayload<SelectWrap<Prisma.tagSelect>>,
    any,
    Prisma.tagSelect,
    Prisma.tagWhereInput,
    false,
    false
> => ({
    validateMap: { __typename: 'Tag' },
    isTransferable: false,
    maxObjects: {
        User: 10000,
        Organization: 0,
    },
    permissionsSelect: () => ({ id: true }),
    permissionResolvers: () => [],
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

const mutater = (): Mutater<
    Tag,
    { graphql: TagCreateInput, db: Prisma.tagUpsertArgs['create'] },
    { graphql: TagUpdateInput, db: Prisma.tagUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, true),
        update: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, false),
    },
    yup: { create: tagsCreate, update: tagsUpdate },
})

const displayer = (): Displayer<
    Prisma.tagSelect,
    Prisma.tagGetPayload<SelectWrap<Prisma.tagSelect>>
> => ({
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