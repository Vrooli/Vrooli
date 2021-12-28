import { Organization, Resource, Routine, RoutineInput, RoutineSearchInput, RoutineSortBy, Tag, User } from "../schema/types";
import { RecursivePartial } from "types";
import { addJoinTables, BaseState, creater, deleter, FormatConverter, MODEL_TYPES, removeJoinTables, reporter, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type RoutineRelationshipList = 'inputs' | 'outputs' | 'nodes' | 'contextualResources' |
    'externalResources' | 'donationResources' | 'tags' | 'users' | 'organizations' | 'starredBy' | 
    'parent' | 'forks' | 'nodeLists' | 'reports' | 'comments';
// Type 2. QueryablePrimitives
export type RoutineQueryablePrimitives = Omit<Routine, RoutineRelationshipList>;
// Type 3. AllPrimitives
export type RoutineAllPrimitives = RoutineQueryablePrimitives;
// type 4. FullModel
export type RoutineFullModel = RoutineAllPrimitives &
Pick<Routine, 'nodes' | 'reports' | 'comments' | 'inputs' | 'outputs' | 'parent'> &
{
    contextualResources: { resource: Resource[] }[],
    externalResources: { resource: Resource[] }[],
    donationResources: { resource: Resource[] }[],
    tags: { tag: Tag[] }[],
    users: { user: User[] }[],
    organizations: { organization: Organization[] }[],
    starredBy: { user: User[] }[],
    forks: { fork: Routine[] }[],
    nodeLists: { list: Routine[] }[],
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
 const formatter = (): FormatConverter<Routine, any> => {
    const joinMapper = {
        contextualResources: 'resource',
        externalResources: 'resource',
        donationResources: 'resource',
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
        starredBy: 'user',
        forks: 'fork',
        nodeLists: 'list',
    };
    return {
        toDB: (obj: RecursivePartial<Routine>): RecursivePartial<any> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Routine> => removeJoinTables(obj, joinMapper)
    }
}

/**
 * Component for search filters
 */
 const sorter = (): Sortable<RoutineSortBy> => ({
    defaultSort: RoutineSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [RoutineSortBy.AlphabeticalAsc]: { name: 'asc' },
            [RoutineSortBy.AlphabeticalDesc]: { name: 'desc' },
            [RoutineSortBy.CommentsAsc]: { comments: { count: 'asc' } },
            [RoutineSortBy.CommentsDesc]: { comments: { count: 'desc' } },
            [RoutineSortBy.ForksAsc]: { forks: { count: 'asc' } },
            [RoutineSortBy.ForksDesc]: { forks: { count: 'desc' } },
            [RoutineSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [RoutineSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [RoutineSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [RoutineSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [RoutineSortBy.StarsAsc]: { stars: { count: 'asc' } },
            [RoutineSortBy.StarsDesc]: { stars: { count: 'desc' } },
            [RoutineSortBy.VotesAsc]: { votes: { count: 'asc' } },
            [RoutineSortBy.VotesDesc]: { votes: { count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { title: { ...insensitive } },
                { description: { ...insensitive } },
                { instructions: { ...insensitive } },
                { tags: { tag: { name: { ...insensitive } } } },
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

export function RoutineModel(prisma?: any) {
    let obj: BaseState<Routine, RoutineFullModel> = {
        prisma,
        model: MODEL_TYPES.Routine,
    }

    return {
        ...obj,
        ...creater<RoutineInput, RoutineFullModel>(obj),
        ...deleter(obj),
        ...formatter(),
        ...reporter(),
        ...searcher<RoutineSortBy, RoutineSearchInput, Routine, RoutineFullModel>(obj),
        ...sorter(),
        ...updater<RoutineInput, RoutineFullModel>(obj),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================