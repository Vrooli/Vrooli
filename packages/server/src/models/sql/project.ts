import { projectsCreate, projectsUpdate, projectTranslationCreate, projectTranslationUpdate } from "@shared/validation";
import { CODE } from "@shared/consts";
import { omit } from '@shared/utils';
import { CustomError } from "../../error";
import { PrismaType, RecursivePartial } from "../../types";
import { Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy, Count, ResourceListUsedFor, ProjectPermission, OrganizationPermission, VisibilityType } from "../../schema/types";
import { addCountFieldsHelper, addCreatorField, addJoinTablesHelper, addOwnerField, addSupplementalFieldsHelper, combineQueries, CUDInput, CUDResult, exceptionsBuilder, FormatConverter, getSearchStringQueryHelper, modelToGraphQL, onlyValidIds, Permissioner, permissionsCheck, removeCountFieldsHelper, removeCreatorField, removeJoinTablesHelper, removeOwnerField, Searcher, selectHelper, validateMaxObjects, ValidateMutationsInput, validateObjectOwnership, visibilityBuilder } from "./base";
import { OrganizationModel, organizationQuerier } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { WalletModel } from "./wallet";
import { genErrorCode } from "../../logger";
import { ViewModel } from "./view";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { tags: 'tag', users: 'user', organizations: 'organization', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', reportsCount: 'reports' };
const supplementalFields = ['isUpvoted', 'isStarred', 'isViewed', 'permissionsProject'];
export const projectFormatter = (): FormatConverter<Project, ProjectPermission> => ({
    relationshipMap: {
        '__typename': 'Project',
        'comments': 'Comment',
        'creator': {
            'User': 'User',
            'Organization': 'Organization',
        },
        'forks': 'Project',
        'owner': {
            'User': 'User',
            'Organization': 'Organization',
        },
        'parent': 'Project',
        'reports': 'Report',
        'resourceLists': 'ResourceList',
        'routines': 'Routine',
        'starredBy': 'User',
        'tags': 'Tag',
        'wallets': 'Wallet',
    },
    constructUnions: (data) => {
        let modified = addCreatorField(data);
        modified = addOwnerField(modified);
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = removeCreatorField(partial);
        modified = removeOwnerField(modified);
        return modified;
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    removeSupplementalFields: (partial) => omit(partial, supplementalFields),
    async addSupplementalFields({ objects, partial, permissions, prisma, userId }): Promise<RecursivePartial<Project>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => StarModel.query(prisma).getIsStarreds(userId, ids, 'Project')],
                ['isUpvoted', async (ids) => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Project')],
                ['isViewed', async (ids) => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Project')],
                ['permissionsProject', async () => await ProjectModel.permissions().get({ objects, permissions, prisma, userId })],
            ]
        });
    }
})

export const projectPermissioner = (): Permissioner<ProjectPermission, ProjectSearchInput> => ({
    async get({
        objects,
        permissions,
        prisma,
        userId,
    }) {
        // Initialize result with default permissions
        const result: (ProjectPermission & { id?: string })[] = objects.map((o) => ({
            id: o.id,
            canComment: false, // own || !isPrivate
            canDelete: false, // own 
            canEdit: false, // own
            canReport: true, // !own && !isPrivate
            canStar: false, // own || !isPrivate
            canView: false, // own || !isPrivate
            canVote: false, // own || !isPrivate
        }));
        // Check ownership
        if (userId) {
            // Query for objects owned by user, or an organization they have an admin role in
            const owned = await prisma.project.findMany({
                where: {
                    id: { in: onlyValidIds(objects.map(o => o.id)) },
                    ...this.ownershipQuery(userId),
                },
                select: { id: true },
            })
            // Set permissions for owned objects
            owned.forEach((o) => {
                const index = objects.findIndex((r) => r.id === o.id);
                result[index] = {
                    ...result[index],
                    canComment: true,
                    canDelete: true,
                    canEdit: true,
                    canReport: false,
                    canStar: true,
                    canView: true,
                    canVote: true,
                }
            });
        }
        // Query all objects
        const all = await prisma.project.findMany({
            where: {
                id: { in: onlyValidIds(objects.map(o => o.id)) },
            },
            select: { id: true, isPrivate: true },
        })
        // Set permissions for all objects
        all.forEach((o) => {
            const index = objects.findIndex((r) => r.id === o.id);
            result[index] = {
                ...result[index],
                canComment: result[index].canComment || !o.isPrivate,
                canReport: result[index].canReport === false ? false : !o.isPrivate,
                canStar: result[index].canStar || !o.isPrivate,
                canView: result[index].canView || !o.isPrivate,
                canVote: result[index].canVote || !o.isPrivate,
            }
        });
        // Return result with IDs removed
        result.forEach((r) => delete r.id);
        return result as ProjectPermission[];
    },
    async canSearch({
        input,
        prisma,
        userId
    }) {
        // Check permissions of specified objects TODO need better approach, but this will do for now
        if (input.ids) {
            const permissions = await this.get({ objects: input.ids.map(id => ({ id })), permissions: [], prisma, userId });
            // Check if trying to view hidden objects
            if (!permissions.every(x => x.canView)) return 'none';
            // If you own all data, you have full search permissions
            if (permissions.every(x => x.canEdit)) return 'full';
        }
        // Now check every specified relationship with permissions
        if (input.organizationId) {
            const isMember = await permissionsCheck({
                actions: ['isMember'],
                model: OrganizationModel,
                object: { id: input.organizationId },
                prisma,
                userId,
            })
            if (isMember) return 'full';
        }
        if (input.parentId) {
            const canEdit = await permissionsCheck({
                model: ProjectModel,
                object: { id: input.parentId },
                actions: ['canEdit'],
                prisma,
                userId,
            })
            if (canEdit) return 'full';
        }
        if (input.userId) {
            if (input.userId === userId) return 'full';
        }
        if (input.organizationId || input.userId || input.parentId) return 'public';
        return 'public';
    },
    ownershipQuery: (userId) => ({
        OR: [
            organizationQuerier().hasRoleInOrganizationQuery(userId),
            { user: { id: userId } }
        ]
    }),
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

export const projectMutater = (prisma: PrismaType) => ({
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ProjectCreateInput, ProjectUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0073') });
        // Validate userIds and organizationIds
        await validateObjectOwnership({ userId, createMany, updateMany, deleteMany, prisma, objectType: 'Project' });
        // Validate max projects
        await validateMaxObjects({ userId, createMany, updateMany, deleteMany, prisma, objectType: 'Project', maxCount: 100 });
        if (createMany) {
            projectsCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel.profanityCheck(createMany);
            createMany.forEach(input => TranslationModel.validateLineBreaks(input, ['description'], CODE.LineBreaksDescription));
            // TODO validate handle
        }
        if (updateMany) {
            projectsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel.profanityCheck(updateMany.map(u => u.data));
            for (const input of updateMany) {
                await WalletModel.verify(prisma).verifyHandle('Project', input.where.id, input.data.handle);
                TranslationModel.validateLineBreaks(input.data, ['description'], CODE.LineBreaksDescription);
            }
        }
    },
    /**
     * Performs adds, updates, and deletes of projects. First validates that every action is allowed.
     */
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<ProjectCreateInput, ProjectUpdateInput>): Promise<CUDResult<Project>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        /**
         * Helper function for creating create/update Prisma value
         */
        const createData = async (input: ProjectCreateInput | ProjectUpdateInput): Promise<{ [x: string]: any }> => ({
            id: input.id,
            handle: (input as ProjectUpdateInput).handle ?? undefined,
            isComplete: input.isComplete ?? undefined,
            isPrivate: input.isPrivate ?? undefined,
            completedAt: (input.isComplete === true) ? new Date().toISOString() : (input.isComplete === false) ? null : undefined,
            parentId: (input as ProjectCreateInput)?.parentId ?? undefined,
            resourceLists: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, input, true),
            tags: await TagModel.mutate(prisma).relationshipBuilder(userId, input, 'Project'),
            translations: TranslationModel.relationshipBuilder(userId, input, { create: projectTranslationCreate, update: projectTranslationUpdate }, false),
        })
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                let data = await createData(input);
                // Associate with either organization or user
                if (input.createdByOrganizationId) {
                    data = {
                        ...data,
                        organization: { connect: { id: input.createdByOrganizationId } },
                        createdByOrganization: { connect: { id: input.createdByOrganizationId } },
                    };
                } else {
                    data = {
                        ...data,
                        user: { connect: { id: userId } },
                        createdByUser: { connect: { id: userId } },
                    };
                }
                // Create object
                const currCreated = await prisma.project.create({
                    data,
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                let data = await createData(input.data);
                // Find object
                let object = await prisma.project.findFirst({ where: input.where })
                if (!object)
                    throw new CustomError(CODE.NotFound, 'Project not found', { code: genErrorCode('0078') });
                // Associate with either organization or user. This will remove the association with the other.
                if (input.data.organizationId) {
                    data = {
                        ...data,
                        organization: { connect: { id: input.data.organizationId } },
                        user: { disconnect: true },
                    };
                } else {
                    data = {
                        ...data,
                        user: { connect: { id: userId } },
                        organization: { disconnect: true },
                    };
                }
                // Update object
                const currUpdated = await prisma.project.update({
                    where: input.where,
                    data,
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.project.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
});

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const ProjectModel = ({
    prismaObject: (prisma: PrismaType) => prisma.project,
    format: projectFormatter(),
    mutate: projectMutater,
    permissions: projectPermissioner,
    search: projectSearcher(),
})

//==============================================================
/* #endregion Model */
//==============================================================