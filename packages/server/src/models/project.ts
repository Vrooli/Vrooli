import { projectsCreate, projectsUpdate, projectTranslationCreate, projectTranslationUpdate } from "@shared/validation";
import { ResourceListUsedFor } from "@shared/consts";
import { addCountFieldsHelper, addJoinTablesHelper, combineQueries, exceptionsBuilder, permissionsSelectHelper, removeCountFieldsHelper, removeJoinTablesHelper, visibilityBuilder } from "./builder";
import { organizationQuerier } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { ViewModel } from "./view";
import { Project, ProjectPermission, ProjectSearchInput, ProjectCreateInput, ProjectUpdateInput, ProjectSortBy } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";
import { Trigger } from "../events";
import { oneIsPublic } from "./utils";
import { isOwnerAdminCheck } from "./validators/isOwnerAdminCheck";
import { getSingleTypePermissions } from "./validators";

const joinMapper = { tags: 'tag', users: 'user', organizations: 'organization', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsProject';
export const projectFormatter = (): FormatConverter<Project, SupplementalFields> => ({
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
            ['isStarred', async () => StarModel.query(prisma).getIsStarreds(userData?.id, ids, 'Project')],
            ['isUpvoted', async () => await VoteModel.query(prisma).getIsUpvoteds(userData?.id, ids, 'Project')],
            ['isViewed', async () => await ViewModel.query(prisma).getIsVieweds(userData?.id, ids, 'Project')],
            ['permissionsProject', async () => await getSingleTypePermissions('Project', ids, prisma, userData)],
        ],
    },
})

export const projectSearcher = (): Searcher<
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

export const projectValidator = (): Validator<
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
    permissionsSelect: (userId) => ({
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
                ], userId)
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
    isAdmin: (data, userId) => isOwnerAdminCheck(data, (d) => (d.root as any).organization, (d) => (d.root as any).user, userId),
    isDeleted: (data) => data.isDeleted || data.root.isDeleted,
    isPublic: (data) => data.isPrivate === false && oneIsPublic<Prisma.projectSelect>(data, [
        ['organization', 'Organization'],
        ['user', 'User'],
    ]),
    ownerOrMemberWhere: (userId) => ({
        root: {
            OR: [
                organizationQuerier().hasRoleInOrganizationQuery(userId),
                { user: { id: userId } }
            ]
        }
    }),
    // createMany.forEach(input => lineBreaksCheck(input, ['description'], CODE.LineBreaksDescription));
    // for (const input of updateMany) {
})


export const projectMutater = (prisma: PrismaType): Mutater<Project> => ({
    async shapeBase(userId: string, data: ProjectCreateInput | ProjectUpdateInput) {
        return {
            id: data.id,
            isComplete: data.isComplete ?? undefined,
            isPrivate: data.isPrivate ?? undefined,
            completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
            permissions: JSON.stringify({}),
            resourceList: await ResourceListModel.mutate(prisma).relationshipBuilder!(userId, data, true),
            tags: await TagModel.mutate(prisma).tagRelationshipBuilder(userId, data, 'Project'),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: projectTranslationCreate, update: projectTranslationUpdate }, false),
        }
    },
    async shapeCreate(userId: string, data: ProjectCreateInput): Promise<Prisma.projectUpsertArgs['create']> {
        return {
            ...this.shapeBase(userId, data),
            parentId: data.parentId ?? undefined,
            organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
            createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
            createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
            user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
        }
    },
    async shapeUpdate(userId: string, data: ProjectUpdateInput): Promise<Prisma.projectUpsertArgs['update']> {
        return {
            ...this.shapeBase(userId, data),
            organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
            user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
        }
    },
    async cud(params: CUDInput<ProjectCreateInput, ProjectUpdateInput>): Promise<CUDResult<Project>> {
        return cudHelper({
            ...params,
            objectType: 'Project',
            prisma,
            yup: { yupCreate: projectsCreate, yupUpdate: projectsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Project', c.id as string, params.userData.id);
                }
            },
        })
    },
});

export const ProjectModel = ({
    prismaObject: (prisma: PrismaType) => prisma.project,
    format: projectFormatter(),
    mutate: projectMutater,
    search: projectSearcher(),
    type: 'Project' as GraphQLModelType,
    validate: projectValidator(),
})