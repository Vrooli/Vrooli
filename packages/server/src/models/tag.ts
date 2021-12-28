import { Organization, Project, Routine, Standard, Tag, TagInput, TagSearchInput, TagSortBy, User } from "../schema/types";
import { RecursivePartial } from "types";
import { addJoinTables, BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, removeJoinTables, reporter, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type TagRelationshipList = 'organizations' | 'projects' | 'routines' | 'standards' |
    'starredBy' | 'votes';
// Type 2. QueryablePrimitives
export type TagQueryablePrimitives = Omit<Tag, TagRelationshipList>;
// Type 3. AllPrimitives
export type TagAllPrimitives = TagQueryablePrimitives;
// type 4. FullModel
export type TagFullModel = TagAllPrimitives &
{
    organizations: { tagged: Organization[] },
    projects: { tagged: Project[] },
    routines: { tagged: Routine[] },
    standards: { tagged: Standard[] },
    starredBy: { user: User[] },
    votes: { voter: User[] },
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
 const formatter = (): FormatConverter<Tag, TagFullModel> => {
    const joinMapper = {
        organizations: 'tagged',
        projects: 'tagged',
        routines: 'tagged',
        standards: 'tagged',
        starredBy: 'user',
        votes: 'voter',
    };
    return {
        toDB: (obj: RecursivePartial<Tag>): RecursivePartial<TagFullModel> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<TagFullModel>): RecursivePartial<Tag> => removeJoinTables(obj, joinMapper)
    }
}

/**
 * Component for search filters
 */
 const sorter = (): Sortable<TagSortBy> => ({
    defaultSort: TagSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [TagSortBy.AlphabeticalAsc]: { name: 'asc' },
            [TagSortBy.AlphabeticalDesc]: { name: 'desc' },
            [TagSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [TagSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [TagSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [TagSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [TagSortBy.StarsAsc]: { stars: { count: 'asc' } },
            [TagSortBy.StarsDesc]: { stars: { count: 'desc' } },
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

export function TagModel(prisma?: any) {
    let obj: BaseState<Tag, TagFullModel> = {
        prisma,
        model: MODEL_TYPES.Tag,
    }

    return {
        ...obj,
        ...creater<TagInput, TagFullModel>(obj),
        ...deleter(obj),
        ...findByIder<TagFullModel>(obj),
        ...formatter(),
        ...reporter(),
        ...searcher<TagSortBy, TagSearchInput, Tag, TagFullModel>(obj),
        ...sorter(),
        ...updater<TagInput, TagFullModel>(obj),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================