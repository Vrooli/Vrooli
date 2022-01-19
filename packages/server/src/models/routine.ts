import { FindByIdInput, Organization, Resource, Routine, RoutineCountInput, RoutineInput, RoutineSearchInput, RoutineSortBy, Tag, User } from "../schema/types";
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
        userId: string | null,
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Create selector
        const select = selectHelper<Routine, RoutineDB>(info, formatter().toDB);
        // Access database
        let routine = await prisma.routine.findUnique({ where: { id: input.id }, ...select });
        // Return routine with "isUpvoted" field. This must be queried separately.
        if (!userId || !routine) return routine;
        const vote = await prisma.vote.findFirst({ where: { userId, routineId: routine.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        return { ...routine, isUpvoted };
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
        // Compute "isUpvoted" field for each routine
        // If userId not provided, then "isUpvoted" is null
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isUpvotedArray = await prisma.vote.findMany({ where: { userId, routineId: { in: resultIds } } });
        console.log('isUpvotedArray', isUpvotedArray)
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            console.log('ids', node.id, isUpvotedArray.map(({ routineId }) => routineId));
            const isUpvoted = isUpvotedArray.find(({ routineId }) => routineId === node.id)?.isUpvote ?? null;
            return { cursor, node: { ...node, isUpvoted } };
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
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(input.organizationId, userId);
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
        // Return routine with "isUpvoted" field. Will be false in this case.
        return { ...routine, isUpvoted: false };
    },
    async updateRoutine(
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
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(input.organizationId, userId);
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
                OR: [
                    { organizationId: input.organizationId },
                    { userId },
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
        // Return routine with "isUpvoted" field. This must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, routineId: routine.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        return { ...routine, isUpvoted };
    },
    async deleteRoutine(userId: string, input: any): Promise<boolean> {
        // Find routine
        const routine = await prisma.routine.findFirst({
            where: {
                OR: [
                    { organizationId: input.organizationId },
                    { userId },
                ]
            }
        })
        if (!routine) throw new CustomError(CODE.ErrorUnknown);
        // Make sure the user is an admin of the organization
        const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(input.organizationId, userId);
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        // Delete comment
        await prisma.routine.delete({
            where: { id: routine.id },
        });
        return true;
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
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Routine>): RecursivePartial<RoutineDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Remove isUpvoted, as it is calculated in its own query
            if (modified.isUpvoted) delete modified.isUpvoted;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<RoutineDB>): RecursivePartial<Routine> => {
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