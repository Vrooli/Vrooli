import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Post, PostCreateInput, PostSearchInput, PostSortBy, PostUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, tagShapeHelper } from "../utils";
import { ModelLogic } from "./types";
import { postValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

const __typename = 'Post' as const;

const suppFields = [] as const;

export const PostModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: PostCreateInput,
    GqlUpdate: PostUpdateInput,
    GqlModel: Post,
    GqlSearch: PostSearchInput,
    GqlSort: PostSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.postUpsertArgs['create'],
    PrismaUpdate: Prisma.postUpsertArgs['update'],
    PrismaModel: Prisma.postGetPayload<SelectWrap<Prisma.postSelect>>,
    PrismaSelect: Prisma.postSelect,
    PrismaWhere: Prisma.postWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.post,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            owner: {
                user: 'User',
                organization: 'Organization',
            },
            reports: 'Report',
            repostedFrom: 'Post',
            reposts: 'Post',
            resourceList: 'ResourceList',
            bookmarkedBy: 'User',
            tags: 'Tag',
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
            repostedFrom: 'Post',
            reposts: 'Post',
            resourceList: 'ResourceList',
            comments: 'Comment',
            bookmarkedBy: 'User',
            votedBy: 'Vote',
            viewedBy: 'View',
            reports: 'Report',
            tags: 'Tag',
        },
        countFields: {
            commentsCount: true,
            repostsCount: true,
        },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                isPinned: noNull(data.isPinned),
                isPrivate: noNull(data.isPrivate),
                organization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
                user: !data.organizationConnect ? { connect: { id: userData.id } } : undefined,
                ...(await shapeHelper({ relation: 'repostedFrom', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Post', parentRelationshipName: 'reposts', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'post', data, prisma, userData })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Post', relation: 'tags', data, prisma, userData })),
            }),
            update: async ({ data, prisma, userData }) => ({
                isPinned: noNull(data.isPinned),
                isPrivate: noNull(data.isPrivate),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Update'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'post', data, prisma, userData })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Post', relation: 'tags', data, prisma, userData })),
            })
        },
        yup: postValidation,
    },
    search: {
        defaultSort: PostSortBy.ScoreDesc,
        sortBy: PostSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            isPinned: true,
            maxScore: true,
            maxBookmarks: true,
            minScore: true,
            minBookmarks: true,
            organizationId: true,
            userId: true,
            repostedFromIds: true,
            tags: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'descriptionWrapped',
                'nameWrapped',
            ]
        }),
    },
    validate: {} as any,
})