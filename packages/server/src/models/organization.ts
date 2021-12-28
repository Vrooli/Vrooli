import { Organization, OrganizationInput, OrganizationSearchInput, OrganizationSortBy, Project, Resource, Routine, Tag, User } from "../schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type OrganizationRelationshipList = 'comments' | 'resources' | 'wallets' | 'projects' | 'starredBy' |
    'routines' | 'tags' | 'reports';
// Type 2. QueryablePrimitives
export type OrganizationQueryablePrimitives = Omit<Organization, OrganizationRelationshipList>;
// Type 3. AllPrimitives
export type OrganizationAllPrimitives = OrganizationQueryablePrimitives;
// type 4. FullModel
export type OrganizationFullModel = OrganizationAllPrimitives &
    Pick<Organization, 'comments' | 'wallets' | 'reports'> &
{
    resources: { resource: Resource[] }[],
    projects: { project: Project[] }[],
    starredBy: { user: User[] }[],
    routines: { routine: Routine[] }[],
    tags: { tag: Tag[] }[],
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
const formatter = (): FormatConverter<any, any> => ({
    toDB: (obj: any): any => ({ ...obj }),
    toGraphQL: (obj: any): any => ({ ...obj })
})

/**
 * Component for search filters
 */
const sorter = (): Sortable<OrganizationSortBy> => ({
    defaultSort: OrganizationSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [OrganizationSortBy.AlphabeticalAsc]: { name: 'asc' },
            [OrganizationSortBy.AlphabeticalDesc]: { name: 'desc' },
            [OrganizationSortBy.CommentsAsc]: { comments: { count: 'asc' } },
            [OrganizationSortBy.CommentsDesc]: { comments: { count: 'desc' } },
            [OrganizationSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [OrganizationSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [OrganizationSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [OrganizationSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [OrganizationSortBy.StarsAsc]: { stars: { count: 'asc' } },
            [OrganizationSortBy.StarsDesc]: { stars: { count: 'desc' } },
            [OrganizationSortBy.VotesAsc]: { votes: { count: 'asc' } },
            [OrganizationSortBy.VotesDesc]: { votes: { count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { name: { ...insensitive } },
                { description: { ...insensitive } },
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

export function OrganizationModel(prisma?: any) {
    let obj: BaseState<Organization, OrganizationFullModel> = {
        prisma,
        model: MODEL_TYPES.Organization,
        formatter: formatter(),
        sorter: sorter(),
    }

    return {
        ...obj,
        ...creater<OrganizationInput, OrganizationFullModel>(obj),
        ...deleter(obj),
        ...findByIder<OrganizationFullModel>(obj),
        ...formatter(),
        ...reporter(),
        ...searcher<OrganizationSortBy, OrganizationSearchInput, Organization, OrganizationFullModel>(obj),
        ...sorter(),
        ...updater<OrganizationInput, OrganizationFullModel>(obj),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================