import { projectValidation } from "@shared/validation";
import { BookmarkModel } from "./bookmark";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Project, ProjectSearchInput, ProjectCreateInput, ProjectUpdateInput, ProjectSortBy, ProjectYou, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { OrganizationModel } from "./organization";
import { getSingleTypePermissions } from "../validators";
import { defaultPermissions, labelShapeHelper, onCommonRoot, oneIsPublic, ownerShapeHelper, preShapeRoot, tagShapeHelper } from "../utils";
import { ProjectVersionModel } from "./projectVersion";
import { SelectWrap } from "../builders/types";
import { getLabels } from "../getters";
import { rootObjectDisplay } from "../utils/rootObjectDisplay";
import { noNull, shapeHelper } from "../builders";

const __typename = 'Project' as const;
type Permissions = Pick<ProjectYou, 'canDelete' | 'canUpdate' | 'canBookmark' | 'canTransfer' | 'canRead' | 'canVote'>;
const suppFields = ['you', 'translatedName'] as const;
export const ProjectModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: ProjectCreateInput,
    GqlUpdate: ProjectUpdateInput,
    GqlModel: Project,
    GqlSearch: ProjectSearchInput,
    GqlSort: ProjectSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.projectUpsertArgs['create'],
    PrismaUpdate: Prisma.projectUpsertArgs['update'],
    PrismaModel: Prisma.projectGetPayload<SelectWrap<Prisma.projectSelect>>,
    PrismaSelect: Prisma.projectSelect,
    PrismaWhere: Prisma.projectWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project,
    display: rootObjectDisplay(ProjectVersionModel),
    format: {
        gqlRelMap: {
            __typename,
            createdBy: 'User',
            issues: 'Issue',
            labels: 'Label',
            owner: {
                ownedByUser: 'User',
                ownedByOrganization: 'Organization',
            },
            parent: 'Project',
            pullRequests: 'PullRequest',
            questions: 'Question',
            quizzes: 'Quiz',
            bookmarkedBy: 'User',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'ProjectVersion',
        },
        prismaRelMap: {
            __typename,
            createdBy: 'User',
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
            parent: 'ProjectVersion',
            issues: 'Issue',
            labels: 'Label',
            tags: 'Tag',
            versions: 'ProjectVersion',
            bookmarkedBy: 'User',
            pullRequests: 'PullRequest',
            stats: 'StatsProject',
            questions: 'Question',
            transfers: 'Transfer',
            quizzes: 'Quiz',
        },
        joinMap: { labels: 'label', bookmarkedBy: 'user', tags: 'tag' },
        countFields: {
            issuesCount: true,
            labelsCount: true,
            pullRequestsCount: true,
            questionsCount: true,
            quizzesCount: true,
            transfersCount: true,
            versionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    },
                    translatedName: await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'project.translatedName')
                }
            },
        },
    },
    mutate: {
        shape: {
            pre: async (params) => {
                const maps = await preShapeRoot({ ...params, objectType: __typename });
                return { ...maps }
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                handle: noNull(data.handle),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions) ?? JSON.stringify({}),
                createdBy: rest.userData?.id ? { connect: { id: rest.userData.id } } : undefined,
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: 'ownedBy', relTypes: ['Connect'], parentRelationshipName: 'projects', isCreate: true, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: 'parent', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ProjectVersion', parentRelationshipName: 'forks', data, ...rest })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersion', parentRelationshipName: 'root', data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Project', relation: 'tags', data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Project', relation: 'labels', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                handle: noNull(data.handle),
                isPrivate: noNull(data.isPrivate),
                permissions: noNull(data.permissions),
                ...rest.preMap[__typename].versionMap[data.id],
                ...(await ownerShapeHelper({ relation: 'ownedBy', relTypes: ['Connect'], parentRelationshipName: 'projects', isCreate: false, objectType: __typename, data, ...rest })),
                ...(await shapeHelper({ relation: 'versions', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersion', parentRelationshipName: 'root', data, ...rest })),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Project', relation: 'tags', data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Project', relation: 'labels', data, ...rest })),
            })
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonRoot({ ...params, objectType: __typename });
            },
        },
        yup: projectValidation,
    },
    search: {
        defaultSort: ProjectSortBy.ScoreDesc,
        sortBy: ProjectSortBy,
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
                { versions: { some: 'transDescriptionWrapped' } },
                { versions: { some: 'transNameWrapped' } },
            ]
        })
    },
    validate: {
        hasCompleteVersion: (data) => data.hasCompleteVersion === true,
        hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
        isDeleted: (data) => data.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<Prisma.projectSelect>(data, [
                ['ownedByOrganization', 'Organization'],
                ['ownedByUser', 'User'],
            ], languages),
        isTransferable: true,
        maxObjects: MaxObjects[__typename],
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
            versions: ['ProjectVersion', ['root']],
        }),
        visibility: {
            private: { isPrivate: true, isDeleted: false },
            public: { isPrivate: false, isDeleted: false },
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})