import { CODE, MemberRole, omit, projectsCreate, projectsUpdate, projectTranslationCreate, projectTranslationUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { PrismaType, RecursivePartial } from "types";
import { Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy, Count, ResourceListUsedFor } from "../../schema/types";
import { addCountFieldsHelper, addCreatorField, addJoinTablesHelper, addOwnerField, CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, PartialGraphQLInfo, removeCountFieldsHelper, removeCreatorField, removeJoinTablesHelper, removeOwnerField, Searcher, selectHelper, ValidateMutationsInput } from "./base";
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
const calculatedFields = ['isUpvoted', 'isStarred', 'role'];
export const projectFormatter = (): FormatConverter<Project> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Project,
        'comments': GraphQLModelType.Comment,
        'creator': {
            'User': GraphQLModelType.User,
            'Organization': GraphQLModelType.Organization,
        },
        'forks': GraphQLModelType.Project,
        'owner': {
            'User': GraphQLModelType.User,
            'Organization': GraphQLModelType.Organization,
        },
        'parent': GraphQLModelType.Project,
        'reports': GraphQLModelType.Report,
        'resourceLists': GraphQLModelType.ResourceList,
        'routines': GraphQLModelType.Routine,
        'starredBy': GraphQLModelType.User,
        'tags': GraphQLModelType.Tag,
        'wallets': GraphQLModelType.Wallet,
    },
    removeCalculatedFields: (partial) => {
        return omit(partial, calculatedFields);
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
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialGraphQLInfo,
    ): Promise<RecursivePartial<Project>[]> {
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, GraphQLModelType.Project)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, GraphQLModelType.Project)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for isViewed
        if (partial.isViewed) {
            const isViewedArray = userId
                ? await ViewModel(prisma).getIsVieweds(userId, ids, GraphQLModelType.Project)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isViewed: isViewedArray[i] }));
        }
        // Query for role
        if (partial.role) {
            let organizationIds: string[] = [];
            // Collect owner data
            let ownerData: any = objects.map(x => x.owner).filter(x => x);
            // If no owner data was found, then owner data was not queried. In this case, query for owner data.
            if (ownerData.length === 0) {
                const ownerDataUnformatted = await prisma.project.findMany({
                    where: { id: { in: ids } },
                    select: {
                        id: true,
                        user: { select: { id: true } },
                        organization: { select: { id: true } },
                    },
                });
                organizationIds = ownerDataUnformatted.map(x => x.organization?.id).filter(x => Boolean(x)) as string[];
                // Inject owner data into "objects"
                objects = objects.map((x, i) => { 
                    const unformatted = ownerDataUnformatted.find(y => y.id === x.id);
                    return ({ ...x, owner: unformatted?.user || unformatted?.organization })
                });
            } else {
                organizationIds = objects
                    .filter(x => Array.isArray(x.owner?.translations) && x.owner.translations.length > 0 && x.owner.translations[0].name)
                    .map(x => x.owner.id)
                    .filter(x => Boolean(x)) as string[];
            }
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.owner?.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: (Boolean(x.owner?.id) && x.owner?.id === userId) ? MemberRole.Owner : undefined };
            }) as any;
        }
        // Convert Prisma objects to GraphQL objects
        return objects as RecursivePartial<Project>[];
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
            TranslationModel().profanityCheck(createMany);
            createMany.forEach(input => TranslationModel().validateLineBreaks(input, ['description'], CODE.LineBreaksDescription));
            // Add createdByOrganizationIds to organizationIds array, if they are set
            organizationIds.push(...createMany.map(input => input.createdByOrganizationId).filter(id => id))
            // Check if user will pass max projects limit
            const existingCount = await prisma.project.count({
                where: {
                    OR: [
                        { user: { id: userId } },
                        { organization: { members: { some: { userId: userId, role: MemberRole.Owner as any } } } },
                    ]
                }
            })
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.MaxProjectsReached, 'Reached the maximum number of projects allowed on this account', { code: genErrorCode('0074') });
            }
        }
        if (updateMany) {
            projectsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel().profanityCheck(updateMany.map(u => u.data));
            // Add new organizationIds to organizationIds array, if they are set
            organizationIds.push(...updateMany.map(input => input.data.organizationId).filter(id => id))
            // Add existing organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.project.findMany({
                where: { id: { in: updateMany.map(input => input.where.id) } },
                select: { id: true, userId: true, organizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.userId !== userId).map(object => object.organizationId));
            for (const input of updateMany) {
                await WalletModel(prisma).verifyHandle(GraphQLModelType.Project, input.where.id, input.data.handle);
                TranslationModel().validateLineBreaks(input.data, ['description'], CODE.LineBreaksDescription);
            }
        }
        if (deleteMany) {
            // Add organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.project.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, userId: true, organizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.userId !== userId).map(object => object.organizationId));
        }
        // Find admin/owner member data for every organization
        const memberData = await OrganizationModel(prisma).isOwnerOrAdmin(userId, organizationIds);
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
            completedAt: input.isComplete ? new Date().toISOString() : null,
            parentId: input.parentId,
            resourceLists: await ResourceListModel(prisma).relationshipBuilder(userId, input, true),
            tags: await TagModel(prisma).relationshipBuilder(userId, input, GraphQLModelType.Project),
            translations: TranslationModel().relationshipBuilder(userId, input, { create: projectTranslationCreate, update: projectTranslationUpdate }, false),
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

export function ProjectModel(prisma: PrismaType) {
    const prismaObject = prisma.project;
    const format = projectFormatter();
    const search = projectSearcher();
    const mutate = projectMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================