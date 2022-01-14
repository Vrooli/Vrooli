import { Organization, Resource, Routine, RoutineCountInput, RoutineInput, RoutineSearchInput, RoutineSortBy, Tag, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountQueries, addJoinTables, counter, creater, deleter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, reporter, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type RoutineRelationshipList = 'inputs' | 'outputs' | 'nodes' | 'contextualResources' |
    'externalResources' | 'tags' | 'users' | 'organizations' | 'starredBy' | 
    'parent' | 'forks' | 'nodeLists' | 'reports' | 'comments';
// Type 2. QueryablePrimitives
export type RoutineQueryablePrimitives = Omit<Routine, RoutineRelationshipList>;
// Type 3. AllPrimitives
export type RoutineAllPrimitives = RoutineQueryablePrimitives;
// type 4. Database shape
export type RoutineDB = RoutineAllPrimitives &
Pick<Routine, 'nodes' | 'reports' | 'comments' | 'inputs' | 'outputs' | 'parent'> &
{
    contextualResources: { resource: Resource[] }[],
    externalResources: { resource: Resource[] }[],
    tags: { tag: Tag[] }[],
    users: { user: User[] }[],
    organizations: { organization: Organization[] }[],
    starredBy: { user: User[] }[],
    forks: { fork: Routine[] }[],
    nodeLists: { list: Routine[] }[],
    _count: { starredBy: number }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<Routine, RoutineDB> => {
    const joinMapper = {
        contextualResources: 'resource',
        externalResources: 'resource',
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
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
    model: keyof PrismaType, 
    toDB: FormatConverter<Routine, RoutineDB>['toDB'],
    toGraphQL: FormatConverter<Routine, RoutineDB>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    async search(where: { [x: string]: any }, input: RoutineSearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        // Many-to-many search queries
        const userIdQuery = input.userId ? { users: { some: { userId: input.userId } } } : {};
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        // One-to-many search queries
        const parentIdQuery = input.parentId ? { forks: { some: { forkId: input.parentId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const search = searcher<RoutineSortBy, RoutineSearchInput, Routine, RoutineDB>(model, toDB, toGraphQL, sorter, prisma);
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
        ...creater<RoutineInput, Routine, RoutineDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...reporter(),
        ...routineSearcher(model, format.toDB, format.toGraphQL, sort, prisma),
        ...updater<RoutineInput, Routine, RoutineDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================