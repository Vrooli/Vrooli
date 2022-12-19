import { projectValidation } from "@shared/validation";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Project, ProjectSearchInput, ProjectCreateInput, ProjectUpdateInput, ProjectSortBy, SessionUser, RootPermission } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Validator, Mutater, Displayer, ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { Trigger } from "../events";
import { OrganizationModel } from "./organization";
import { getSingleTypePermissions } from "../validators";
import { noNull, padSelect, permissionsSelectHelper } from "../builders";
import { oneIsPublic } from "../utils";
import { ProjectVersionModel } from "./projectVersion";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: ProjectCreateInput,
    GqlUpdate: ProjectUpdateInput,
    GqlModel: Project,
    GqlSearch: ProjectSearchInput,
    GqlSort: ProjectSortBy,
    GqlPermission: RootPermission,
    PrismaCreate: Prisma.projectUpsertArgs['create'],
    PrismaUpdate: Prisma.projectUpsertArgs['update'],
    PrismaModel: Prisma.projectGetPayload<SelectWrap<Prisma.projectSelect>>,
    PrismaSelect: Prisma.projectSelect,
    PrismaWhere: Prisma.projectWhereInput,
}

const __typename = 'Project' as const;

const suppFields = ['isStarred', 'isUpvoted', 'isViewed', 'permissionsRoot'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
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
        starredBy: 'User',
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
        starredBy: 'User',
        pullRequests: 'PullRequest',
        stats: 'StatsProject',
        questions: 'Question',
        transfers: 'Transfer',
        quizzes: 'Quiz',
    },
    joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
    countFields: ['issuesCount', 'labelsCount', 'pullRequestsCount', 'questionsCount', 'quizzesCount', 'versionsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => ({
            isStarred: async () => StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
            isUpvoted: async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
            isViewed: async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
            permissionsRoot: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
        }),
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: ProjectSortBy.ScoreDesc,
    sortBy: ProjectSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'hasCompleteVersion',
        'labelsId',
        'maxScore',
        'maxStars',
        'minScore',
        'minStars',
        'ownedByOrganizationId',
        'ownedByUserId',
        'parentId',
        'tags',
        'translationLanguagesLatestVersion',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: () => ({
        OR: [
            'tagsWrapped',
            'labelsWrapped',
            { versions: { some: 'transDescriptionWrapped' } },
            { versions: { some: 'transNameWrapped' } },
        ]
    })
})

const validator = (): Validator<Model> => ({
    isTransferable: true,
    hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
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
    permissionsSelect: (...params) => ({
        id: true,
        hasCompleteVersion: true,
        isDeleted: true,
        isPrivate: true,
        permissions: true,
        createdBy: 'User',
        ownedByOrganization: 'Organization',
        ownedByUser: 'User',
        versions: 'ProjectVersion',
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ([
        // ['canComment', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canDelete', async () => isAdmin && !isDeleted],
        // ['canEdit', async () => isAdmin && !isDeleted],
        // ['canReport', async () => !isAdmin && !isDeleted && isPublic],
        // // ['canRun', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canStar', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canView', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canVote', async () => !isDeleted && (isAdmin || isPublic)],
    ] as any),
    owner: (data) => ({
        Organization: data.ownedByOrganization,
        User: data.ownedByUser,
    }),
    hasCompletedVersion: (data) => data.hasCompleteVersion === true,
    isDeleted: (data) => data.isDeleted,// || data.root.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false && oneIsPublic<Prisma.projectSelect>(data, [
        ['ownedByOrganization', 'Organization'],
        ['ownedByUser', 'User'],
    ], languages),
    visibility: {
        private: {
            isPrivate: true,
            // OR: [
            //     { isPrivate: true },
            //     { root: { isPrivate: true } },
            // ]
        },
        public: {
            isPrivate: false,
            // AND: [
            //     { isPrivate: false },
            //     { root: { isPrivate: false } },
            // ]
        },
        owner: (userId) => ({
            OR: [
                { ownedByUser: { id: userId } },
                { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            ]
            // root: {
            //     OR: [
            //         { ownedByUser: { id: userId } },
            //         { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            //     ]
            // }
        }),
    }
    // createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksDescription'));
    // for (const input of updateMany) {
})

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


const mutater = (): Mutater<Model> => ({
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
        onUpdated: ({ updated, prisma, userData }) => {
            // for (const u of updated) {
            //     Trigger(prisma, userData.languages).updateProject(userData.id, u.id as string);
            // }
        }
    },
    yup: projectValidation,
});

const displayer = (): Displayer<Model> => ({
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
})

export const ProjectModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
    search: searcher(),
    validate: validator(),
})