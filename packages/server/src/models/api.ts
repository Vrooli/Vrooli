import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Api, ApiCreateInput, ApiSearchInput, ApiSortBy, ApiUpdateInput, ApiYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { StarModel } from "./bookmark";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";
import { defaultPermissions, labelShapeHelper, tagShapeHelper } from "../utils";
import { noNull, shapeHelper } from "../builders";
import { apiValidation } from "@shared/validation";
import { OrganizationModel } from "./organization";

const __typename = 'Api' as const;
type Permissions = Pick<ApiYou, 'canDelete' | 'canUpdate' | 'canBookmark' | 'canTransfer' | 'canRead' | 'canVote'>;
const suppFields = ['you'] as const;
export const ApiModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: ApiCreateInput,
    GqlUpdate: ApiUpdateInput,
    GqlModel: Api,
    GqlPermission: Permissions,
    GqlSearch: ApiSearchInput,
    GqlSort: ApiSortBy,
    PrismaCreate: Prisma.apiUpsertArgs['create'],
    PrismaUpdate: Prisma.apiUpsertArgs['update'],
    PrismaModel: Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>,
    PrismaSelect: Prisma.apiSelect,
    PrismaWhere: Prisma.apiWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api,
    display: {
        select: () => ({
            id: true,
            versions: {
                orderBy: { versionIndex: 'desc' },
                take: 1,
                select: ApiVersionModel.display.select(),
            }
        }),
        label: (select, languages) => select.versions.length > 0 ?
            ApiVersionModel.display.label(select.versions[0] as any, languages) : '',
    },
    format: {
        gqlRelMap: {
            __typename,
            createdBy: 'User',
            owner: {
                ownedByUser: 'User',
                ownedByOrganization: 'Organization',
            },
            parent: 'Api',
            tags: 'Tag',
            versions: 'ApiVersion',
            labels: 'Label',
            issues: 'Issue',
            pullRequests: 'PullRequest',
            questions: 'Question',
            bookmarkedBy: 'User',
            stats: 'StatsApi',
            transfers: 'Transfer',
        },
        prismaRelMap: {
            __typename,
            createdBy: 'User',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            parent: 'ApiVersion',
            tags: 'Tag',
            issues: 'Issue',
            bookmarkedBy: 'User',
            votedBy: 'Vote',
            viewedBy: 'View',
            pullRequests: 'PullRequest',
            versions: 'ApiVersion',
            labels: 'Label',
            stats: 'StatsApi',
            questions: 'Question',
            transfers: 'Transfer',
        },
        joinMap: { labels: 'label', bookmarkedBy: 'user', tags: 'tag' },
        countFields: {
            issuesCount: true,
            pullRequestsCount: true,
            questionsCount: true,
            transfersCount: true,
            versionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await StarModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ prisma, userData, data }) => ({
                id: data.id,
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: userData?.id ? { connect: { id: userData.id } } : undefined,
                ...(await shapeHelper({ relation: 'ownedByUser', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'User', parentRelationshipName: 'apisCreated', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'ownedByOrganization', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName: 'apis', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'parent', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ApiVersion', parentRelationshipName: 'forks', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'ApiVersion', parentRelationshipName: 'root', data, prisma, userData })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Api', relation: 'tags', data, prisma, userData })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Api', relation: 'labels', data, prisma, userData })),

            }),
            update: async ({ prisma, userData, data }) => ({
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                createdBy: userData?.id ? { connect: { id: userData.id } } : undefined,
                ...(await shapeHelper({ relation: 'ownedByUser', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'User', parentRelationshipName: 'apisCreated', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'ownedByOrganization', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName: 'apis', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'ApiVersion', parentRelationshipName: 'root', data, prisma, userData })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Api', relation: 'tags', data, prisma, userData })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Api', relation: 'labels', data, prisma, userData })),
            }),
        },
        yup: apiValidation,
    },
    search: {
        defaultSort: ApiSortBy.ScoreDesc,
        sortBy: ApiSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            hasCompleteVersion: true,
            issuesId: true,
            labelsIds: true,
            maxScore: true,
            maxBookmarks: true,
            maxViews: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            parentId: true,
            pullRequestsId: true,
            tags: true,
            translationLanguagesLatestVersion: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'tagsWrapped',
                'labelsWrapped',
                { versions: { some: 'transNameWrapped' } },
                { versions: { some: 'transSummaryWrapped' } }
            ]
        })
    },
    validate: {
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        isTransferable: true,
        maxObjects: {
            User: {
                private: {
                    noPremium: 1,
                    premium: 10,
                },
                public: {
                    noPremium: 3,
                    premium: 100,
                }
            },
            Organization: {
                private: {
                    noPremium: 3,
                    premium: 25,
                },
                public: {
                    noPremium: 5,
                    premium: 100,
                }
            },
        },
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isPrivate: true,
            permissions: true,
            createdBy: 'User',
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
            versions: ['ApiVersion', ['root']],
        }),
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})