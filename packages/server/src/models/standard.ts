import { PrismaType, RecursivePartial } from "types";
import { Routine, Standard, StandardCountInput, StandardInput, StandardSearchInput, StandardSortBy, Tag, User } from "../schema/types";
import { addCountQueries, addJoinTables, counter, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, removeCountQueries, removeJoinTables, reporter, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type StandardRelationshipList = 'tags' | 'routineInputs' | 'routineOutputs' | 'starredBy' |
    'reports' | 'comments';
// Type 2. QueryablePrimitives
export type StandardQueryablePrimitives = Omit<Standard, StandardRelationshipList>;
// Type 3. AllPrimitives
export type StandardAllPrimitives = StandardQueryablePrimitives;
// type 4. FullModel
export type StandardFullModel = StandardAllPrimitives &
    Pick<Standard, 'reports' | 'comments'> &
{
    tags: { tag: Tag[] }[],
    routineInputs: { routine: Routine[] }[],
    routineOutputs: { routine: Routine[] }[],
    starredBy: { user: User[] }[],
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
const formatter = (): FormatConverter<Standard, StandardFullModel> => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Standard>): RecursivePartial<StandardFullModel> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<StandardFullModel>): RecursivePartial<Standard> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<StandardSortBy> => ({
    defaultSort: StandardSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [StandardSortBy.AlphabeticalAsc]: { name: 'asc' },
            [StandardSortBy.AlphabeticalDesc]: { name: 'desc' },
            [StandardSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [StandardSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [StandardSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [StandardSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [StandardSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [StandardSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [StandardSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [StandardSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
            [StandardSortBy.VotesAsc]: { votes: { _count: 'asc' } },
            [StandardSortBy.VotesDesc]: { votes: { _count: 'desc' } },
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

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StandardModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Standard;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<StandardCountInput>(model, prisma),
        ...creater<StandardInput, Standard, StandardFullModel>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Standard, StandardFullModel>(model, format.toDB, prisma),
        ...reporter(),
        ...searcher<StandardSortBy, StandardSearchInput, Standard, StandardFullModel>(model, format.toDB, format.toGraphQL, sort, prisma),
        ...updater<StandardInput, Standard, StandardFullModel>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================