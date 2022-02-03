import { CODE, projectAdd, projectUpdate } from "@local/shared";
import { CustomError } from "../error";
import { GraphQLResolveInfo } from "graphql";
import { PrismaType, RecursivePartial } from "types";
import { DeleteOneInput, FindByIdInput, Organization, Project, ProjectCountInput, ProjectAddInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy, Resource, Success, Tag, User } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, addOwnerField, counter, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { ResourceModel } from "./resource";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type ProjectRelationshipList = 'resources' | 'wallets' | 'user' | 'organization' | 'createdByUser' | 'createdByOrganization' | 'starredBy' |
    'parent' | 'forks' | 'reports' | 'tags' | 'comments' | 'routines';
// Type 2. QueryablePrimitives
export type ProjectQueryablePrimitives = Omit<Project, ProjectRelationshipList>;
// Type 3. AllPrimitives
export type ProjectAllPrimitives = ProjectQueryablePrimitives;
// type 4. Database shape
export type ProjectDB = ProjectAllPrimitives &
    Pick<Omit<Project, 'creator' | 'owner'>, 'wallets' | 'reports' | 'comments' | 'routines'> &
{
    user: User;
    organization: Organization;
    createdByUser: User;
    createdByOrganization: Organization;
    resources: { resource: Resource }[],
    starredBy: { user: User }[],
    parent: { project: Project }[],
    forks: { project: Project }[],
    tags: { tag: Tag }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Custom component for creating project. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
const projectCreater = (toDB: FormatConverter<Project, ProjectDB>['toDB'], prisma: PrismaType) => ({
    async create(
        data: any,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<ProjectDB> | null> {
        // Remove any relationships should not be created/connected in this operation
        data = keepOnly(data, ['resources', 'parent', 'tags']);
        // Perform additional checks
        // TODO
        // Create
        const { id } = await prisma.project.create({ data });
        // Query database
        return await prisma.user.findUnique({ where: { id }, ...selectHelper<Project, ProjectDB>(info, toDB) }) as RecursivePartial<ProjectDB> | null;
    }
})

/**
 * Component for formatting between graphql and prisma types
 */
export const projectFormatter = (): FormatConverter<Project, ProjectDB> => {
    const joinMapper = {
        resources: 'resource',
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
        starredBy: 'user',
        forks: 'fork',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Project>): RecursivePartial<ProjectDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Remove isUpvoted and isStarred, as they are calculated in their own queries
            if (modified.isUpvoted) delete modified.isUpvoted;
            if (modified.isStarred) delete modified.isStarred;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<ProjectDB>): RecursivePartial<Project> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            modified = addCreatorField(modified);
            modified = addOwnerField(modified);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
export const projectSorter = (): Sortable<ProjectSortBy> => ({
    defaultSort: ProjectSortBy.DateUpdatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ProjectSortBy.AlphabeticalAsc]: { name: 'asc' },
            [ProjectSortBy.AlphabeticalDesc]: { name: 'desc' },
            [ProjectSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [ProjectSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [ProjectSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [ProjectSortBy.ForksDesc]: { forks: { _count: 'desc' } },
            [ProjectSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ProjectSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ProjectSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ProjectSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ProjectSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [ProjectSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
            [ProjectSortBy.VotesAsc]: { score: 'asc' },
            [ProjectSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { name: { ...insensitive } },
                { description: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    }
})

/**
 * Handles the authorized adding, updating, and deleting of projects.
 */
const projecter = (format: FormatConverter<Project, ProjectDB>, sort: Sortable<ProjectSortBy>, prisma: PrismaType) => ({
    async findProject(
        userId: string | null | undefined, // Of the user making the request, not the project
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Create selector
        const select = selectHelper<Project, ProjectDB>(info, format.toDB);
        // Access database
        let project = await prisma.project.findUnique({ where: { id: input.id }, ...select });
        // Return project with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!project) throw new CustomError(CODE.InternalError, 'Project not found');
        if (!userId) return { ...project, isUpvoted: false, isStarred: false };
        const vote = await prisma.vote.findFirst({ where: { userId, projectId: project.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, projectId: project.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...project, isUpvoted, isStarred };
    },
    async searchProjects(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: ProjectSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const parentIdQuery = input.parentId ? { forks: { some: { forkId: input.parentId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        // Search
        const search = searcher<ProjectSortBy, ProjectSearchInput, Project, ProjectDB>(MODEL_TYPES.Project, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where }, input, info);
        // Compute "isUpvoted" and "isStarred" fields for each project
        // If userId not provided, then "isUpvoted" is null and "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null, isStarred: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isUpvotedArray = await prisma.vote.findMany({ where: { userId, projectId: { in: resultIds } } });
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, projectId: { in: resultIds } } });
        console.log('isUpvotedArray', isUpvotedArray);
        console.log('isStarredArray', isStarredArray);
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            console.log('ids', node.id, isUpvotedArray.map(({ projectId }) => projectId));
            const isUpvoted = isUpvotedArray.find(({ projectId }) => projectId === node.id)?.isUpvote ?? null;
            const isStarred = Boolean(isStarredArray.find(({ projectId }) => projectId === node.id));   
            console.log('isStarred', isStarred, );
            return { cursor, node: { ...node, isUpvoted, isStarred } };
        });
        return searchResults;
    },
    async addProject(
        userId: string,
        input: ProjectAddInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        projectAdd.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.description)) throw new CustomError(CODE.BannedWord);
        // Create project data
        let projectData: { [x: string]: any } = { name: input.name, description: input.description ?? '' };
        // Associate with either organization or user
        if (input.createdByOrganizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            projectData = { 
                ...projectData, 
                organization: { connect: { id: input.createdByOrganizationId } },
                createdByOrganization: { connect: { id: input.createdByOrganizationId } },
            };
        } else {
            projectData = { 
                ...projectData, 
                user: { connect: { id: userId } },
                createdByUser: { connect: { id: userId } },
            };
        }
        // Handle resources
        const resourceData = ResourceModel(prisma).relationshipBuilder(userId, input, true);
        if (resourceData) projectData.resources = resourceData;
        // Create project
        const project = await prisma.project.create({
            data: projectData as any,
            ...selectHelper<Project, ProjectDB>(info, format.toDB)
        })
        // Return project with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...project, isUpvoted: null, isStarred: false };
    },
    async updateProject(
        userId: string,
        input: ProjectUpdateInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        projectUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.description)) throw new CustomError(CODE.BannedWord);
        // Create project data
        let projectData: { [x: string]: any } = { name: input.name, description: input.description ?? '' };
        // Associate with either organization or user. This will remove the association with the other.
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            projectData = { ...projectData, organization: { connect: { id: input.organizationId } }, userId: null };
        } else {
            projectData = { ...projectData, user: { connect: { id: userId } }, organizationId: null };
        }
        // Handle resources
        const resourceData = ResourceModel(prisma).relationshipBuilder(userId, input, false);
        if (resourceData) projectData.resources = resourceData;
        // Find project
        let project = await prisma.project.findFirst({
            where: {
                AND: [
                    { id: input.id },
                    { OR: [
                        { organizationId: input.organizationId },
                        { userId },
                    ] }
                ]
            }
        })
        if (!project) throw new CustomError(CODE.ErrorUnknown);
        // Update project
        project = await prisma.project.update({
            where: { id: project.id },
            data: projectData as any,
            ...selectHelper<Project, ProjectDB>(info, format.toDB)
        });
        // Return project with "isUpvoted" and "isStarred" fields. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, projectId: project.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, projectId: project.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...project, isUpvoted, isStarred };
    },
    async deleteProject(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const project = await prisma.project.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                userId: true,
                organizationId: true,
            }
        })
        if (!project) throw new CustomError(CODE.NotFound, "Project not found");
        // Check if user is authorized
        let authorized = userId === project.userId;
        if (!authorized && project.organizationId) {
            authorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, project.organizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.project.delete({
            where: { id: project.id },
        });
        return { success: true };
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ProjectModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Project;
    const format = projectFormatter();
    const sort = projectSorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<ProjectCountInput>(model, prisma),
        ...projecter(format, sort, prisma),
        ...projectCreater(format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================