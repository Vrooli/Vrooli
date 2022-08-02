import { CODE, omit, projectsCreate, projectsUpdate, projectTranslationCreate, projectTranslationUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { PrismaType, RecursivePartial } from "../../types";
import { Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy, Count, ResourceListUsedFor, ProjectPermission, OrganizationPermission } from "../../schema/types";
import { addCountFieldsHelper, addCreatorField, addJoinTablesHelper, addOwnerField, addSupplementalFieldsHelper, CUDInput, CUDResult, FormatConverter, modelToGraphQL, Permissioner, permissionsCheck, removeCountFieldsHelper, removeCreatorField, removeJoinTablesHelper, removeOwnerField, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { OrganizationModel } from "./organization";
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
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
    addCountFields: (partial) => {
        return addCountFieldsHelper(partial, countMapper);
    },
    removeCountFields: (data) => {
        return removeCountFieldsHelper(data, countMapper);
    },
    removeSupplementalFields: (partial) => {
        return omit(partial, supplementalFields);
    },
    async addSupplementalFields({ objects, partial, permissions, prisma, userId }): Promise<RecursivePartial<Project>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => StarModel.query(prisma).getIsStarreds(userId, ids, 'Project')],
                ['isUpvoted', async (ids) => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Project')],
                ['isViewed', async (ids) => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Project')],
                ['permissionsProject', async () => await ProjectModel.permissions(prisma).get({ objects, permissions, userId })],
            ]
        });
    }
})

export const projectPermissioner = (prisma: PrismaType): Permissioner<ProjectPermission, ProjectSearchInput> => ({
    async get({
        objects,
        permissions,
        userId,
    }) {
        // Initialize result with ID
        const result: Partial<ProjectPermission>[] = objects.map((o) => ({
            canComment: true,
            canDelete: false,
            canEdit: false,
            canReport: true,
            canStar: true,
            canVote: true,
            canView: true,
        }));
        const ids = objects.map(x => x.id);
        let ownerData: { 
            id: string, 
            user?: { id: string } | null | undefined, 
            organization?: { id: string } | null | undefined
        }[] = [];
        // If some owner data missing, query for owner data.
        if (objects.map(x => x.owner).filter(x => x).length < objects.length) {
            ownerData = await prisma.project.findMany({
                where: { id: { in: ids } },
                select: {
                    id: true,
                    user: { select: { id: true } },
                    organization: { select: { id: true } },
                },
            });
        } else {
            ownerData = objects.map((x) => {
                const isOrg = Boolean(Array.isArray(x.owner?.translations) && x.owner.translations.length > 0 && x.owner.translations[0].name);
                return ({
                    id: x.id,
                    user: isOrg ? null : x.owner,
                    organization: isOrg ? x.owner : null,
                });
            });
        }
        // Find permissions for every organization
        const organizationIds: string[] = ownerData.map(x => x.organization?.id).filter(x => Boolean(x)) as string[];
        const orgPermissions = await OrganizationModel.permissions(prisma).get({ 
            objects: organizationIds.map(x => ({ id: x })),
            userId 
        });
        // Find which objects have ownership permissions
        for (let i = 0; i < objects.length; i++) {
            const unformatted = ownerData.find(y => y.id === objects[i].id);
            if (!unformatted) continue;
            // Check if user owns object directly, or through organization
            if (unformatted.user?.id !== userId) {
                const orgIdIndex = organizationIds.findIndex(id => id === unformatted?.organization?.id);
                if (orgIdIndex < 0) continue;
                if (!orgPermissions[orgIdIndex].canEdit) continue;
            }
            // Set owner permissions
            result[i].canDelete = true;
            result[i].canEdit = true;
            result[i].canView = true;
        }
        // TODO isPrivate view check
        // TODO check relationships for permissions
        return result as ProjectPermission[];
    },
    async canSearch({
        input,
        userId
    }) {
        // Check permissions of specified objects TODO need better approach, but this will do for now
        if (input.ids) {
            const permissions = await this.get({ objects: input.ids.map(id => ({ id })), permissions: [], userId });
            // Check if trying to view hidden objects
            if (input.includePrivate === true && !permissions.every(x => x.canView)) return 'none';
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
            if (!isMember) return 'none';
        }
        if (input.parentId) {
            const canEdit = await permissionsCheck({
                model: ProjectModel,
                object: { id: input.parentId },
                actions: ['canEdit'],
                prisma,
                userId,
            })
            if (!canEdit) return 'none';
        }
        if (input.userId) {
            if (input.userId !== userId) return 'none';
        }
        if (input.organizationId || input.userId || input.parentId) return 'full';
        return 'public';
    }
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
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    },
    customQueries(input: ProjectSearchInput): { [x: string]: any } {
        // isComplete routines may be set to true or false generally, and also set exceptions
        let isComplete: any;
        if (!!input.isCompleteExceptions) {
            isComplete = { OR: [{ isComplete: input.isComplete }] };
            for (const exception of input.isCompleteExceptions) {
                if (['createdByOrganization', 'createdByUser', 'organization', 'project', 'user'].includes(exception.relation)) {
                    isComplete.OR.push({ [exception.relation]: { id: exception.id } });
                }
            }
        } else {
            isComplete = { isComplete: input.isComplete };
        }
        return {
            ...isComplete,
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            ...(input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            ...(input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            ...(input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            ...(input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            ...(input.userId !== undefined ? { userId: input.userId } : {}),
            ...(input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            ...(input.parentId !== undefined ? { parentId: input.parentId } : {}),
            ...(input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            ...(input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        }
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
        // Collect organizationIds from each object, and check if the user is an admin/owner of every organization
        const organizationIds: (string | null | undefined)[] = [];
        if (createMany) {
            projectsCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel.profanityCheck(createMany);
            createMany.forEach(input => TranslationModel.validateLineBreaks(input, ['description'], CODE.LineBreaksDescription));
            // Add createdByOrganizationIds to organizationIds array, if they are set
            organizationIds.push(...createMany.map(input => input.createdByOrganizationId).filter(id => id))
            // Check if user will pass max projects limit
            //TODO
            // const existingCount = await prisma.project.count({
            //     where: {
            //         OR: [
            //             { user: { id: userId } },
            //             { organization: { members: { some: { userId: userId, role: MemberRole.Owner as any } } } },
            //         ]
            //     }
            // })
            // if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
            //     throw new CustomError(CODE.MaxProjectsReached, 'Reached the maximum number of projects allowed on this account', { code: genErrorCode('0074') });
            // }
            // TODO handle
        }
        if (updateMany) {
            projectsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel.profanityCheck(updateMany.map(u => u.data));
            // Add new organizationIds to organizationIds array, if they are set
            organizationIds.push(...updateMany.map(input => input.data.organizationId).filter(id => id))
            // Add existing organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.project.findMany({
                where: { id: { in: updateMany.map(input => input.where.id) } },
                select: { id: true, userId: true, organizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.userId !== userId).map(object => object.organizationId));
            for (const input of updateMany) {
                await WalletModel.verify(prisma).verifyHandle('Project', input.where.id, input.data.handle);
                TranslationModel.validateLineBreaks(input.data, ['description'], CODE.LineBreaksDescription);
            }
        }
        if (deleteMany) {
            const objects = await prisma.project.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, userId: true, organizationId: true },
            });
            // Split objects by userId and organizationId
            const userIds = objects.filter(object => Boolean(object.userId)).map(object => object.userId);
            if (userIds.some(id => id !== userId))
                throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.', { code: genErrorCode('0243') })
            // Add to organizationIds array, to check ownership status
            organizationIds.push(...objects.filter(object => !userId.includes(object.organizationId ?? '')).map(object => object.organizationId));
        }
        // Find admin/owner member data for every organization
        const memberData = await OrganizationModel.query(prisma).isOwnerOrAdmin(userId, organizationIds);
        // If any member data is undefined, the user is not authorized to delete one or more objects
        if (memberData.some(member => !member))
            throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.', { code: genErrorCode('0076') })
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
            handle: (input as ProjectUpdateInput).handle ?? null,
            isComplete: input.isComplete,
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