import { PrismaType, RecursivePartial } from "../types";
import { Organization, OrganizationCountInput, OrganizationInput, OrganizationSearchInput, OrganizationSortBy, Project, Resource, Routine, Tag, User } from "../schema/types";
import { addJoinTables, counter, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, removeJoinTables, reporter, searcher, Sortable, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type OrganizationRelationshipList = 'comments' | 'resources' | 'wallets' | 'projects' | 'starredBy' |
    'routines' | 'tags' | 'reports' | 'donationResources';
// Type 2. QueryablePrimitives
export type OrganizationQueryablePrimitives = Omit<Organization, OrganizationRelationshipList>;
// Type 3. AllPrimitives
export type OrganizationAllPrimitives = OrganizationQueryablePrimitives;
// type 4. FullModel
export type OrganizationFullModel = OrganizationAllPrimitives &
    Pick<Organization, 'comments' | 'wallets' | 'reports'> &
{
    resources: { resource: Resource[] }[],
    donationResources: { resource: Resource[] }[],
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
 const formatter = (): FormatConverter<Organization, OrganizationFullModel> => {
    const joinMapper = {
        donationResources: 'resource',
        resources: 'resource',
        projects: 'project',
        starredBy: 'user',
        routines: 'routine',
        tags: 'tag',
    };
    return {
        toDB: (obj: RecursivePartial<Organization>): RecursivePartial<OrganizationFullModel> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<OrganizationFullModel>): RecursivePartial<Organization> => removeJoinTables(obj, joinMapper)
    }
}

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

export function OrganizationModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Organization;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<OrganizationCountInput>(model, prisma),
        ...creater<OrganizationInput, Organization, OrganizationFullModel>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Organization, OrganizationFullModel>(model, format.toDB, prisma),
        ...reporter(),
        ...searcher<OrganizationSortBy, OrganizationSearchInput, Organization, OrganizationFullModel>(model, format.toDB, format.toGraphQL, sort, prisma),
        ...updater<OrganizationInput, Organization, OrganizationFullModel>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================