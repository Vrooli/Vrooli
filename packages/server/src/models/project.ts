import { projectsCreate, projectsUpdate, projectTranslationCreate, projectTranslationUpdate } from "@shared/validation";
import { ResourceListUsedFor } from "@shared/consts";
import { addCountFieldsHelper, addJoinTablesHelper, combineQueries, exceptionsBuilder, getSearchStringQueryHelper, permissionsSelectHelper, removeCountFieldsHelper, removeJoinTablesHelper, visibilityBuilder } from "./builder";
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
import { getPermissions, oneIsPublic } from "./utils";
import { isOwnerAdminCheck } from "./validators/isOwnerAdminCheck";

const joinMapper = { tags: 'tag', users: 'user', organizations: 'organization', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsProject';
export const projectFormatter = (): FormatConverter<Project, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Project',
        comments: 'Comment',
        creator: {
            User: 'User',
            Organization: 'Organization',
        },
        forks: 'Project',
        owner: {
            User: 'User',
            Organization: 'Organization',
        },
        parent: 'Project',
        reports: 'Report',
        resourceLists: 'ResourceList',
        routines: 'Routine',
        starredBy: 'User',
        tags: 'Tag',
        wallets: 'Wallet',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isStarred', 'isUpvoted', 'isViewed', 'permissionsProject'],
        toGraphQL: ({ ids, prisma, userId }) => [
            ['isStarred', async () => StarModel.query(prisma).getIsStarreds(userId, ids, 'Project')],
            ['isUpvoted', async () => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Project')],
            ['isViewed', async () => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Project')],
            ['permissionsProject', async () => await getPermissions({ objectType: 'Project', ids, prisma, userId })],
        ],
    },
})

export const projectSearcher = (): Searcher<ProjectSearchInput> => ({
    defaultSort: ProjectSortBy.VotesDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ProjectSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [ProjectSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [ProjectSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [ProjectSortBy.ForksDesc]: { forks: { _count: 'desc' } },
            [ProjectSortBy.DateCompletedAsc]: { completedAt: 'asc' },
            [ProjectSortBy.DateCompletedDesc]: { completedAt: 'desc' },
            [ProjectSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ProjectSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ProjectSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ProjectSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ProjectSortBy.StarsAsc]: { stars: 'asc' },
            [ProjectSortBy.StarsDesc]: { stars: 'desc' },
            [ProjectSortBy.VotesAsc]: { score: 'asc' },
            [ProjectSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
                    { tags: { some: { tag: { tag: { ...insensitive } } } } },
                ]
            })
        })
    },
    customQueries(input: ProjectSearchInput, userId: string | null | undefined): { [x: string]: any } {
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
    Prisma.projectGetPayload<{ select: { [K in keyof Required<Prisma.projectSelect>]: true } }>,
    ProjectPermission,
    Prisma.projectSelect,
    Prisma.projectWhereInput
> => ({
    validateMap: {
        __typename: 'Project',
        user: 'User',
        organization: 'Organization',
        parent: 'Project',
    },
    permissionsSelect: (userId) => ({
        id: true,
        isComplete: true,
        isPrivate: true,
        permissions: true,
        ...permissionsSelectHelper([
            ['organization', 'Organization'],
            ['user', 'User'],
        ], userId)
    }),
    permissionResolvers: (data, userId) => {
        const isAdmin = userId && projectValidator().isAdmin(data, userId);
        const isPublic = projectValidator().isPublic(data);
        return [
            ['canComment', async () => isAdmin || isPublic],
            ['canDelete', async () => isAdmin],
            ['canEdit', async () => isAdmin],
            ['canReport', async () => !isAdmin && isPublic],
            ['canStar', async () => isAdmin || isPublic],
            ['canView', async () => isAdmin || isPublic],
            ['canVote', async () => isAdmin || isPublic],
        ]
    },
    isAdmin: (data, userId) => isOwnerAdminCheck(data, (d) => d.organization, (d) => d.user, userId),
    isPublic: (data) => data.isPrivate === false && oneIsPublic<Prisma.projectSelect>(data, [
        ['organization', 'Organization'],
        ['user', 'User'],
    ]),
    ownerOrMemberWhere: (userId) => ({
        OR: [
            organizationQuerier().hasRoleInOrganizationQuery(userId),
            { user: { id: userId } }
        ]
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
            resourceList: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, true),
            tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Project'),
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
                    Trigger(prisma).objectCreate('Project', c.id as string, params.userId);
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