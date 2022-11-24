import { projectsCreate, projectsUpdate } from "@shared/validation";
import { ResourceListUsedFor } from "@shared/consts";
import { addCountFieldsHelper, addJoinTablesHelper, combineQueries, exceptionsBuilder, permissionsSelectHelper, removeCountFieldsHelper, removeJoinTablesHelper, visibilityBuilder } from "./builder";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { ViewModel } from "./view";
import { Project, ProjectPermission, ProjectSearchInput, ProjectCreateInput, ProjectUpdateInput, ProjectSortBy, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { Trigger } from "../events";
import { oneIsPublic, translationRelationshipBuilder } from "./utils";
import { getSingleTypePermissions } from "./validators";
import { OrganizationModel } from "./organization";
import { relBuilderHelper } from "./actions";

const joinMapper = { tags: 'tag', users: 'user', organizations: 'organization', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsProject';
const formatter = (): Formatter<Project, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Project',
        comments: 'Comment',
        creator: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        forks: 'Project',
        owner: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        parent: 'Project',
        reports: 'Report',
        resourceLists: 'ResourceList',
        starredBy: 'User',
        tags: 'Tag',
    },
    rootFields: ['hasCompleteVersion', 'isDeleted', 'isInternal', 'isPrivate', 'votes', 'stars', 'views', 'permissions'],
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isStarred', 'isUpvoted', 'isViewed', 'permissionsProject'],
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'Project')],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, 'Project')],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, 'Project')],
            ['permissionsProject', async () => await getSingleTypePermissions('Project', ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<
    ProjectSearchInput,
    ProjectSortBy,
    Prisma.project_versionOrderByWithRelationInput,
    Prisma.project_versionWhereInput
> => ({
    defaultSort: ProjectSortBy.VotesDesc,
    sortMap: {
        CommentsAsc: { comments: { _count: 'asc' } },
        CommentsDesc: { comments: { _count: 'desc' } },
        ForksAsc: { forks: { _count: 'asc' } },
        ForksDesc: { forks: { _count: 'desc' } },
        DateCompletedAsc: { completedAt: 'asc' },
        DateCompletedDesc: { completedAt: 'desc' },
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        StarsAsc: { root: { stars: 'asc' } },
        StarsDesc: { root: { stars: 'desc' } },
        VotesAsc: { root: { votes: 'asc' } },
        VotesDesc: { root: { votes: 'desc' } },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
            { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
            { tags: { some: { tag: { tag: { ...insensitive } } } } },
        ]
    }),
    customQueries(input, userId) {
        const isComplete = exceptionsBuilder({
            canQuery: ['createdByOrganization', 'createdByUser', 'organization.id', 'project.id', 'user.id'],
            exceptionField: 'isCompleteExceptions',
            input,
            mainField: 'isComplete',
        })
        return combineQueries([
            isComplete,
            visibilityBuilder({ model: ProjectModel, userId, visibility: input.visibility }),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            (input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            (input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            (input.userId !== undefined ? { userId: input.userId } : {}),
            (input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            (input.parentId !== undefined ? { parentId: input.parentId } : {}),
            (input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            (input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        ])
    },
})

const validator = (): Validator<
    ProjectCreateInput,
    ProjectUpdateInput,
    Project,
    Prisma.project_versionGetPayload<{ select: { [K in keyof Required<Prisma.project_versionSelect>]: true } }>,
    ProjectPermission,
    Prisma.project_versionSelect,
    Prisma.project_versionWhereInput
> => ({
    validateMap: {
        __typename: 'Project',
        root: {
            select: {
                parent: 'Project',
                organization: 'Organization',
                user: 'User',
            }
        },
        forks: 'Project',
    },
    permissionsSelect: (...params) => ({
        id: true,
        isComplete: true,
        isDeleted: true,
        isPrivate: true,
        root: {
            select: {
                isDeleted: true,
                isPrivate: true,
                isInternal: true,
                permissions: true,
                ...permissionsSelectHelper([
                    ['organization', 'Organization'],
                    ['user', 'User'],
                ], ...params)
            }
        },
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ([
        ['canComment', async () => !isDeleted && (isAdmin || isPublic)],
        ['canDelete', async () => isAdmin && !isDeleted],
        ['canEdit', async () => isAdmin && !isDeleted],
        ['canReport', async () => !isAdmin && !isDeleted && isPublic],
        // ['canRun', async () => !isDeleted && (isAdmin || isPublic)],
        ['canStar', async () => !isDeleted && (isAdmin || isPublic)],
        ['canView', async () => !isDeleted && (isAdmin || isPublic)],
        ['canVote', async () => !isDeleted && (isAdmin || isPublic)],
    ]),
    owner: (data) => ({
        Organization: (data.root as any).organization,
        User: (data.root as any).user,
    }),
    isDeleted: (data) => data.isDeleted || data.root.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false && oneIsPublic<Prisma.projectSelect>(data, [
        ['organization', 'Organization'],
        ['user', 'User'],
    ], languages),
    ownerOrMemberWhere: (userId) => ({
        root: {
            OR: [
                OrganizationModel.query.hasRoleInOrganizationQuery(userId),
                { user: { id: userId } }
            ]
        }
    }),
    // createMany.forEach(input => lineBreaksCheck(input, ['description'], CODE.LineBreaksDescription));
    // for (const input of updateMany) {
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ProjectCreateInput | ProjectUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        isComplete: data.isComplete ?? undefined,
        isPrivate: data.isPrivate ?? undefined,
        completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
        permissions: JSON.stringify({}),
        resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
        tags: await TagModel.mutate(prisma).tagRelationshipBuilder(userData.id, data, 'Project'),
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}


const mutater = (): Mutater<
    Project,
    { graphql: ProjectCreateInput, db: Prisma.projectUpsertArgs['create'] },
    { graphql: ProjectUpdateInput, db: Prisma.projectUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                ...(await shapeBase(prisma, userData, data, true)),
                parentId: data.parentId ?? undefined,
                organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
            }
        },
        update: async ({ data, prisma, userData }) => {
            return {
                ...(await shapeBase(prisma, userData, data, false)),
                organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
            }
        }
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).objectCreate('Project', c.id as string, userData.id);
            }
        },
    },
    yup: { create: projectsCreate, update: projectsUpdate },
});

export const ProjectModel = ({
    delegate: (prisma: PrismaType) => prisma.project,
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'Project' as GraphQLModelType,
    validate: validator(),
})