import { CODE, projectCreate, projectUpdate } from "@local/shared";
import { CustomError } from "../error";
import { PrismaType, RecursivePartial } from "types";
import { DeleteOneInput, FindByIdInput, Project, ProjectCountInput, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy, Success } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, addOwnerField, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { ResourceModel } from "./resource";
import { project } from "@prisma/client";
import { TagModel } from "./tag";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
export const projectFormatter = (): FormatConverter<Project, project> => {
    const joinMapper = {
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Project>): RecursivePartial<project> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Remove isUpvoted and isStarred, as they are calculated in their own queries
            if (modified.isUpvoted) delete modified.isUpvoted;
            if (modified.isStarred) delete modified.isStarred;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<project>): RecursivePartial<Project> => {
            console.log('projectFormatter.toGraphQL start', obj);
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            modified = addCreatorField(modified);
            modified = addOwnerField(modified);
            console.log('projectFormatter.toGraphQL finished', modified);
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
const projecter = (format: FormatConverter<Project, project>, sort: Sortable<ProjectSortBy>, prisma: PrismaType) => ({
    async find(
        userId: string | null | undefined, // Of the user making the request, not the project
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Project> | null> {
        // Create selector
        const select = selectHelper<Project, project>(info, format.toDB);
        // Access database
        let project = await prisma.project.findUnique({ where: { id: input.id }, ...select });
        // Return project with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!project) throw new CustomError(CODE.InternalError, 'Project not found');
        if (!userId) return { ...format.toGraphQL(project), isUpvoted: false, isStarred: false };
        const vote = await prisma.vote.findFirst({ where: { userId, projectId: project.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, projectId: project.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(project), isUpvoted, isStarred };
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: ProjectSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const parentIdQuery = input.parentId ? { parentId: input.parentId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        // Search
        const search = searcher<ProjectSortBy, ProjectSearchInput, Project, project>(MODEL_TYPES.Project, format.toDB, format.toGraphQL, sort, prisma);
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
    async create(
        userId: string,
        input: ProjectCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Project>> {
        // Check for valid arguments
        projectCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.description)) throw new CustomError(CODE.BannedWord);
        // Create project data
        let projectData: { [x: string]: any } = { 
            name: input.name, 
            description: input.description, 
            parentId: input.parentId,
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, true),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, true),
        };
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
        // Create project
        const project = await prisma.project.create({
            data: projectData as any,
            ...selectHelper<Project, project>(info, format.toDB)
        })
        // Return project with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.toGraphQL(project), isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: ProjectUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Project>> {
        // Check for valid arguments
        projectUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.description)) throw new CustomError(CODE.BannedWord);
        // Create project data
        let projectData: { [x: string]: any } = { 
            name: input.name, 
            description: input.description, 
            parentId: input.parentId,
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, false),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, false),
        };
        // Associate with either organization or user. This will remove the association with the other.
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            projectData = { ...projectData, organization: { connect: { id: input.organizationId } }, userId: null };
        } else {
            projectData = { ...projectData, user: { connect: { id: userId } }, organizationId: null };
        }
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
            ...selectHelper<Project, project>(info, format.toDB)
        });
        // Return project with "isUpvoted" and "isStarred" fields. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, projectId: project.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, projectId: project.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(project), isUpvoted, isStarred };
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
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
    }
}

//==============================================================
/* #endregion Model */
//==============================================================