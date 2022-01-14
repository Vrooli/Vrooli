import { PrismaType, RecursivePartial } from "types";
import { Routine, Standard, StandardCountInput, StandardInput, StandardSearchInput, StandardSortBy, Tag, User } from "../schema/types";
import { addCountQueries, addJoinTables, counter, creater, deleter, findByIder, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, reporter, searcher, Sortable, updater } from "./base";

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
// type 4. Database shape
export type StandardDB = StandardAllPrimitives &
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
const formatter = (): FormatConverter<Standard, StandardDB> => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Standard>): RecursivePartial<StandardDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            if (modified.creator) {
                if (modified.creator.hasOwnProperty('username')) {
                    modified.user = modified.creator;
                } else {
                    modified.organization = modified.creator;
                }
                delete modified.creator;
            }
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<StandardDB>): RecursivePartial<Standard> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            if (modified.user) {
                modified.creator = modified.user;
                delete modified.user;
            } else if (modified.organization) {
                modified.creator = modified.organization;
                delete modified.organization;
            }
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

/**
 * Component for searching
 */
 export const standardSearcher = (
    model: keyof PrismaType, 
    toDB: FormatConverter<Standard, StandardDB>['toDB'],
    toGraphQL: FormatConverter<Standard, StandardDB>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    async search(where: { [x: string]: any }, input: StandardSearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        // Many-to-one search queries
        const userIdQuery = input.userId ? { userId: input.userId } : {};
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : {};
        // One-to-many search queries
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const routineIdQuery = input.routineId ? { 
            OR: [
                { routineInputs: { some: { routineId: input.routineId } } },
                { routineOutputs: { some: { routineId: input.routineId } } },
            ]
         } : {};
        const search = searcher<StandardSortBy, StandardSearchInput, Standard, StandardDB>(model, toDB, toGraphQL, sorter, prisma);
        return search.search({...userIdQuery, ...organizationIdQuery, ...reportIdQuery, ...routineIdQuery, ...where}, input, info);
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
        ...creater<StandardInput, Standard, StandardDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Standard, StandardDB>(model, format.toDB, prisma),
        ...reporter(),
        ...standardSearcher(model, format.toDB, format.toGraphQL, sort, prisma),
        ...updater<StandardInput, Standard, StandardDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================