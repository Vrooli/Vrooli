import { Organization, Project, Routine, Standard, Tag, TagCountInput, TagInput, TagSearchInput, TagSortBy, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addJoinTables, counter, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, removeJoinTables, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type TagRelationshipList = 'organizations' | 'projects' | 'routines' | 'standards' |
    'starredBy';
// Type 2. QueryablePrimitives
export type TagQueryablePrimitives = Omit<Tag, TagRelationshipList>;
// Type 3. AllPrimitives
export type TagAllPrimitives = TagQueryablePrimitives;
// type 4. Database shape
export type TagDB = TagAllPrimitives &
{
    organizations: { tagged: Organization },
    projects: { tagged: Project },
    routines: { tagged: Routine },
    standards: { tagged: Standard },
    starredBy: { user: User },
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
 export const tagFormatter = (): FormatConverter<Tag, TagDB> => {
    const joinMapper = {
        organizations: 'tagged',
        projects: 'tagged',
        routines: 'tagged',
        standards: 'tagged',
        starredBy: 'user',
    };
    return {
        toDB: (obj: RecursivePartial<Tag>): RecursivePartial<TagDB> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<TagDB>): RecursivePartial<Tag> => removeJoinTables(obj, joinMapper)
    }
}

/**
 * Component for search filters
 */
 export const tagSorter = (): Sortable<TagSortBy> => ({
    defaultSort: TagSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [TagSortBy.AlphabeticalAsc]: { name: 'asc' },
            [TagSortBy.AlphabeticalDesc]: { name: 'desc' },
            [TagSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [TagSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [TagSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [TagSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [TagSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [TagSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { tag: { ...insensitive } },
                { description: { ...insensitive } },
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

export function TagModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Tag;
    const format = tagFormatter();
    const sort = tagSorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<TagCountInput>(model, prisma),
        ...creater<TagInput, Tag, TagDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Tag, TagDB>(model, format.toDB, prisma),
        ...searcher<TagSortBy, TagSearchInput, Tag, TagDB>(model, format.toDB, format.toGraphQL, sort, prisma),
        ...updater<TagInput, Tag, TagDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================