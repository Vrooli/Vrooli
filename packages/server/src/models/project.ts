import { projectValidation } from "@shared/validation";
import { StarModel } from "./bookmark";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Project, ProjectSearchInput, ProjectCreateInput, ProjectUpdateInput, ProjectSortBy, SessionUser, ProjectYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { Trigger } from "../events";
import { OrganizationModel } from "./organization";
import { getSingleTypePermissions } from "../validators";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ProjectVersionModel } from "./projectVersion";
import { SelectWrap } from "../builders/types";
import { getLabels } from "../getters";

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ProjectCreateInput | ProjectUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        // isComplete: data.isComplete ?? undefined,
        // isPrivate: data.isPrivate ?? undefined,
        // completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
        permissions: JSON.stringify({}),
        // resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
        // tags: await tagRelationshipBuilder(prisma, userData, data, 'Project', isAdd),
        // translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}

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
    display: {
        select: () => ({
            id: true,
            versions: {
                where: { isPrivate: false },
                orderBy: { versionIndex: 'desc' },
                take: 1,
                select: ProjectVersionModel.display.select(),
            }
        }),
        label: (select, languages) => select.versions.length > 0 ?
            ProjectVersionModel.display.label(select.versions[0] as any, languages) : '',
    },
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
                        isBookmarked: await StarModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
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
            create: async ({ data, prisma, userData }) => ({
                // id: data.id,
                // parentId: noNull(data.parentId),
                // createdBy: { connect: { id: userData.id } },
                // ...connectOwner(data, userData),
            } as any),
            update: async ({ data, prisma, userData }) => ({

            } as any)
        },
        trigger: {
            onCreated: ({ created, prisma, userData }) => {
                for (const c of created) {
                    Trigger(prisma, userData.languages).createProject(userData.id, c.id);
                }
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
        maxObjects: {
            User: {
                private: {
                    noPremium: 3,
                    premium: 100,
                },
                public: {
                    noPremium: 25,
                    premium: 250,
                },
            },
            Organization: {
                private: {
                    noPremium: 3,
                    premium: 100,
                },
                public: {
                    noPremium: 25,
                    premium: 250,
                },
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
            versions: ['ProjectVersion', ['root']],
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