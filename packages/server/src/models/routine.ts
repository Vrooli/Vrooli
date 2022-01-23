import { DeleteOneInput, FindByIdInput, Organization, Resource, Routine, RoutineCountInput, RoutineInput, RoutineSearchInput, RoutineSortBy, Success, Tag, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountQueries, addCreatorField, addJoinTables, addOwnerField, counter, deleter, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { GraphQLResolveInfo } from "graphql";
import { CustomError } from "../error";
import { CODE } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type RoutineRelationshipList = 'inputs' | 'outputs' | 'nodes' | 'contextualResources' |
    'externalResources' | 'tags' | 'starredBy' | 'project' | 'user' | 'organization' |
    'createdByUser' | 'createdByOrganization' | 'parent' | 'forks' | 'nodeLists' | 'reports' | 'comments';
// Type 2. QueryablePrimitives
export type RoutineQueryablePrimitives = Omit<Routine, RoutineRelationshipList>;
// Type 3. AllPrimitives
export type RoutineAllPrimitives = RoutineQueryablePrimitives;
// type 4. Database shape
export type RoutineDB = RoutineAllPrimitives &
    Pick<Omit<Routine, 'creator' | 'owner'>, 'nodes' | 'reports' | 'comments' | 'inputs' | 'outputs' | 'parent' | 'project'> &
{
    user: User;
    organization: Organization;
    createdByUser: User;
    createdByOrganization: Organization;
    contextualResources: { resource: Resource }[],
    externalResources: { resource: Resource }[],
    tags: { tag: Tag }[],
    starredBy: { user: User }[],
    forks: { fork: Routine }[],
    nodeLists: { list: Routine }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Custom component for creating routine. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
const routineCreater = (toDB: FormatConverter<Routine, RoutineDB>['toDB'], prisma: PrismaType) => ({
    async create(
        data: any,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<RoutineDB> | null> {
        // Remove any relationships should not be created/connected in this operation
        data = keepOnly(data, ['inputs', 'outputs', 'nodes', 'contextualResources', 'externalResources', 'tags', 'users', 'organizations', 'parent']);
        // Perform additional checks
        // TODO
        // Create
        const { id } = await prisma.routine.create({ data });
        // Query database
        return await prisma.routine.findUnique({ where: { id }, ...selectHelper<Routine, RoutineDB>(info, toDB) }) as RecursivePartial<RoutineDB> | null;
    }
})

/**
 * Handles the authorized adding, updating, and deleting of routines.
 */
const routiner = (format: FormatConverter<Routine, RoutineDB>, sort: Sortable<RoutineSortBy>, prisma: PrismaType) => ({
    async findRoutine(
        userId: string | null, // Of the user making the request, not the routine
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Create selector
        const select = selectHelper<Routine, RoutineDB>(info, formatter().toDB);
        // Access database
        let routine = await prisma.routine.findUnique({ where: { id: input.id }, ...select });
        // Return routine with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!userId || !routine) return routine;
        const vote = await prisma.vote.findFirst({ where: { userId, routineId: routine.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, routineId: routine.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...routine, isUpvoted, isStarred };
    },
    async searchRoutines(
        where: { [x: string]: any },
        userId: string | null,
        input: RoutineSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const parentIdQuery = input.parentId ? { forks: { some: { forkId: input.parentId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        // Search
        const search = searcher<RoutineSortBy, RoutineSearchInput, Routine, RoutineDB>(MODEL_TYPES.Routine, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where }, input, info);
        // Compute "isUpvoted" and "isStarred" field for each routine
        // If userId not provided, then "isUpvoted" is null and "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null, isStarred: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isUpvotedArray = await prisma.vote.findMany({ where: { userId, routineId: { in: resultIds } } });
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, routineId: { in: resultIds } } });
        console.log('isUpvotedArray', isUpvotedArray)
        console.log('isStarredArray', isStarredArray)
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            console.log('ids', node.id, isUpvotedArray.map(({ routineId }) => routineId));
            const isUpvoted = isUpvotedArray.find(({ routineId }) => routineId === node.id)?.isUpvote ?? null;
            const isStarred = Boolean(isStarredArray.find(({ routineId }) => routineId === node.id));
            return { cursor, node: { ...node, isUpvoted, isStarred } };
        });
        return searchResults;
    },
    async addRoutine(
        userId: string,
        input: RoutineInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.title || input.title.length < 1) throw new CustomError(CODE.InternalError, 'Title is too short');
        // Check for censored words
        if (hasProfanity(input.title) || hasProfanity(input.description ?? '')) throw new CustomError(CODE.BannedWord);
        // Create routine data
        let routineData: { [x: string]: any } = {
            name: input.title,
            description: input.description,
            instructions: input.instructions,
            version: input.version,
            isAutomatable: input.isAutomatable,
        };
        // Associate with either organization or user
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            routineData = { 
                ...routineData, 
                organization: { connect: { id: input.organizationId } },
                createdByOrganization: { connect: { id: input.organizationId } },
            };
        } else {
            routineData = { 
                ...routineData, 
                user: { connect: { id: userId } },
                createdByUser: { connect: { id: userId } },
            };
        }
        // TODO inputs
        // TODO outputs
        // TODO resources
        // Create routine
        const routine = await prisma.routine.create({
            data: routineData as any,
            ...selectHelper<Routine, RoutineDB>(info, format.toDB)
        })
        // Return routine with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...routine, isUpvoted: null, isStarred: false };
    },
    async updateRoutine(
        userId: string,
        input: RoutineInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.id) throw new CustomError(CODE.InternalError, 'No routine id provided');
        if (!input.title || input.title.length < 1) throw new CustomError(CODE.InternalError, 'Title is too short');
        // Check for censored words
        if (hasProfanity(input.title) || hasProfanity(input.description ?? '')) throw new CustomError(CODE.BannedWord);
        // Create routine data
        let routineData: { [x: string]: any } = {
            name: input.title,
            description: input.description,
            instructions: input.instructions,
            version: input.version,
            isAutomatable: input.isAutomatable,
        };
        // Associate with either organization or user. This will remove the association with the other.
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            routineData = { ...routineData, organization: { connect: { id: input.organizationId } } };
        } else {
            routineData = { ...routineData, user: { connect: { id: userId } } };
        }
        // TODO inputs
        // TODO outputs
        // TODO resources
        // Find routine
        let routine = await prisma.routine.findFirst({
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
        if (!routine) throw new CustomError(CODE.ErrorUnknown);
        // Update routine
        routine = await prisma.routine.update({
            where: { id: routine.id },
            data: routine as any,
            ...selectHelper<Routine, RoutineDB>(info, format.toDB)
        });
        // Return routine with "isUpvoted" and "isStarred" field. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, routineId: routine.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, routineId: routine.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...routine, isUpvoted, isStarred };
    },
    async deleteRoutine(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const routine = await prisma.routine.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                userId: true,
                organizationId: true,
            }
        })
        if (!routine) throw new CustomError(CODE.NotFound, "Routine not found");
        // Check if user is authorized
        let authorized = userId === routine.userId;
        if (!authorized && routine.organizationId) {
            authorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, routine.organizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.routine.delete({
            where: { id: routine.id },
        });
        return { success: true };
    }
})

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Routine, RoutineDB> => {
    const joinMapper = {
        contextualResources: 'resource',
        externalResources: 'resource',
        tags: 'tag',
        starredBy: 'user',
        forks: 'fork',
        nodeLists: 'list',
    };
    return {
        toDB: (obj: RecursivePartial<Routine>): RecursivePartial<RoutineDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Remove isUpvoted and isStarred, as they are calculated in their own queries
            if (modified.isUpvoted) delete modified.isUpvoted;
            if (modified.isStarred) delete modified.isStarred;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<RoutineDB>): RecursivePartial<Routine> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = addCreatorField(modified);
            modified = addOwnerField(modified);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<RoutineSortBy> => ({
    defaultSort: RoutineSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [RoutineSortBy.AlphabeticalAsc]: { title: 'asc' },
            [RoutineSortBy.AlphabeticalDesc]: { title: 'desc' },
            [RoutineSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [RoutineSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [RoutineSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [RoutineSortBy.ForksDesc]: { forks: { _count: 'desc' } },
            [RoutineSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [RoutineSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [RoutineSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [RoutineSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [RoutineSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [RoutineSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
            [RoutineSortBy.VotesAsc]: { score: 'asc' },
            [RoutineSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { title: { ...insensitive } },
                { description: { ...insensitive } },
                { instructions: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoutineModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Routine;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<RoutineCountInput>(model, prisma),
        ...deleter(model, prisma),
        ...routineCreater(format.toDB, prisma),
        ...routiner(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================