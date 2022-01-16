import { PrismaType, RecursivePartial } from "../types";
import { Organization, OrganizationCountInput, OrganizationSearchInput, OrganizationSortBy, Project, Resource, Routine, Tag, User } from "../schema/types";
import { addCountQueries, addJoinTables, counter, deleter, findByIder, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, reporter, searcher, selectHelper, Sortable } from "./base";
import { GraphQLResolveInfo } from "graphql";
import { CustomError } from "../error";
import { CODE } from "@local/shared";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type OrganizationRelationshipList = 'comments' | 'resources' | 'wallets' | 'projects' | 'starredBy' |
    'routines' | 'routinesCreated' | 'tags' | 'reports';
// Type 2. QueryablePrimitives
export type OrganizationQueryablePrimitives = Omit<Organization, OrganizationRelationshipList>;
// Type 3. AllPrimitives
export type OrganizationAllPrimitives = OrganizationQueryablePrimitives;
// type 4. Database shape
export type OrganizationDB = OrganizationAllPrimitives &
    Pick<Organization, 'comments' | 'wallets' | 'reports' | 'routines' | 'routinesCreated'> &
{
    resources: { resource: Resource }[],
    projects: { project: Project }[],
    starredBy: { user: User }[],
    tags: { tag: Tag }[],
    _count: { starredBy: number }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Custom component for creating organization. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
 const organizationCreater = (toDB: FormatConverter<Organization, OrganizationDB>['toDB'], prisma?: PrismaType) => ({
    async create(
        data: any, 
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<OrganizationDB> | null> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Remove any relationships should not be created/connected in this operation
        data = keepOnly(data, ['resources', 'tags', 'members', 'projects']);
        // Perform additional checks
        // TODO
        // Create
        const { id } = await prisma.organization.create({ data });
        // Query database
        return await prisma.organization.findUnique({ where: { id }, ...selectHelper<Organization, OrganizationDB>(info, toDB) }) as RecursivePartial<OrganizationDB> | null;
    }
})

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<Organization, OrganizationDB> => {
    const joinMapper = {
        resources: 'resource',
        projects: 'project',
        starredBy: 'user',
        tags: 'tag',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Organization>): RecursivePartial<OrganizationDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<OrganizationDB>): RecursivePartial<Organization> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            return modified;
        },
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
            [OrganizationSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [OrganizationSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [OrganizationSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [OrganizationSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [OrganizationSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [OrganizationSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { name: { ...insensitive } },
                { bio: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    }
})

/**
 * Component for searching
 */
 export const organizationSearcher = ( 
    toDB: FormatConverter<Organization, OrganizationDB>['toDB'],
    toGraphQL: FormatConverter<Organization, OrganizationDB>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    async search(where: { [x: string]: any }, input: OrganizationSearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        // Many-to-many search queries
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        // One-to-many search queries
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const userIdQuery = input.userId ? { members: { some: { id: input.userId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        const search = searcher<OrganizationSortBy, OrganizationSearchInput, Organization, OrganizationDB>(MODEL_TYPES.Organization, toDB, toGraphQL, sorter, prisma);
        return search.search({...projectIdQuery, ...routineIdQuery, ...userIdQuery, ...reportIdQuery, ...standardIdQuery, ...where}, input, info);
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
        ...deleter(model, prisma),
        ...findByIder<Organization, OrganizationDB>(model, format.toDB, prisma),
        ...reporter(),
        ...organizationCreater(format.toDB, prisma),
        ...organizationSearcher(format.toDB, format.toGraphQL, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================