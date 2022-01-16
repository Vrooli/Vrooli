import { Organization, Resource, Routine, RoutineCountInput, RoutineInput, RoutineSearchInput, RoutineSortBy, Tag, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountQueries, addJoinTables, counter, creater, deleter, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, reporter, searcher, selectHelper, Sortable, updater } from "./base";
import { GraphQLResolveInfo } from "graphql";
import { CustomError } from "../error";
import { CODE } from "@local/shared";

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
Pick<Routine, 'nodes' | 'reports' | 'comments' | 'inputs' | 'outputs' | 'parent' | 'project' | 'user' | 'organization' | 'createdByUser' | 'createdByOrganization'> &
{
    contextualResources: { resource: Resource }[],
    externalResources: { resource: Resource }[],
    tags: { tag: Tag }[],
    starredBy: { user: User }[],
    forks: { fork: Routine }[],
    nodeLists: { list: Routine }[],
    _count: { starredBy: number }[],
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
 const routineCreater = (toDB: FormatConverter<Routine, RoutineDB>['toDB'], prisma?: PrismaType) => ({
    async create(
        data: any, 
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<RoutineDB> | null> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
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
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<RoutineDB>): RecursivePartial<Routine> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
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
            [RoutineSortBy.VotesAsc]: { votes: { _count: 'asc' } },
            [RoutineSortBy.VotesDesc]: { votes: { _count: 'desc' } },
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

/**
 * Component for searching
 */
 export const routineSearcher = (
    toDB: FormatConverter<Routine, RoutineDB>['toDB'],
    toGraphQL: FormatConverter<Routine, RoutineDB>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    async search(where: { [x: string]: any }, input: RoutineSearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        // One-to-one search queries
        const userIdQuery = input.userId ? { user: { id: input.userId } } : {};
        const organizationIdQuery = input.organizationId ? { organization: { id: input.organizationId } } : {};
        // One-to-many search queries
        const parentIdQuery = input.parentId ? { forks: { some: { forkId: input.parentId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const search = searcher<RoutineSortBy, RoutineSearchInput, Routine, RoutineDB>(MODEL_TYPES.Routine, toDB, toGraphQL, sorter, prisma);
        return search.search({...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where}, input, info);
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoutineModel(prisma?: PrismaType) {
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
        ...reporter(),
        ...routineCreater(format.toDB, prisma),
        ...routineSearcher(format.toDB, format.toGraphQL, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================