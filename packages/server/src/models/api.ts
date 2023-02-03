import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Api, ApiCreateInput, ApiSearchInput, ApiSortBy, ApiUpdateInput, ApiYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { StarModel } from "./star";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";
import { labelShapeHelper, tagShapeHelper } from "../utils";
import { noNull, padSelect, shapeHelper } from "../builders";
import { apiValidation } from "@shared/validation";
import { OrganizationModel } from "./organization";

const __typename = 'Api' as const;
type Permissions = Pick<ApiYou, 'canDelete' | 'canEdit' | 'canStar' | 'canTransfer' | 'canView' | 'canVote'>;
const suppFields = ['you.canDelete', 'you.canEdit', 'you.canStar', 'you.canTransfer', 'you.canView', 'you.canVote', 'you.isStarred', 'you.isUpvoted', 'you.isViewed'] as const;
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
            starredBy: 'User',
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
            starredBy: 'User',
            votedBy: 'Vote',
            viewedBy: 'View',
            pullRequests: 'PullRequest',
            versions: 'ApiVersion',
            labels: 'Label',
            stats: 'StatsApi',
            questions: 'Question',
            transfers: 'Transfer',
        },
        joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
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
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                    'you.isViewed': await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    'you.isUpvoted': await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
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
            maxStars: true,
            maxViews: true,
            minScore: true,
            minStars: true,
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
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            canComment: () => !isDeleted && (isAdmin || isPublic),
            canDelete: () => isAdmin && !isDeleted,
            canEdit: () => isAdmin && !isDeleted,
            canReport: () => !isAdmin && !isDeleted && isPublic,
            canStar: () => !isDeleted && (isAdmin || isPublic),
            canTransfer: () => isAdmin && !isDeleted,
            canView: () => !isDeleted && (isAdmin || isPublic),
            canVote: () => !isDeleted && (isAdmin || isPublic),
        }),
        permissionsSelect: (...params) => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isPrivate: true,
            permissions: true,
            createdBy: 'User',
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
            versions: 'ApiVersion',
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