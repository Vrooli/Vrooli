import { CODE } from "@local/shared";
import { CustomError } from "../error";
import { GraphQLResolveInfo } from "graphql";
import { PrismaType, RecursivePartial } from "types";
import { Organization, Project, ProjectCountInput, ProjectInput, ProjectSearchInput, ProjectSortBy, Resource, Tag, User } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, addOwnerField, counter, creater, deleter, findByIder, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type ProjectRelationshipList = 'resources' | 'wallets' | 'user' | 'organization' | 'createdByUser' | 'createdByOrganization' | 'starredBy' |
    'parent' | 'forks' | 'reports' | 'tags' | 'comments' | 'routines';
// Type 2. QueryablePrimitives
export type ProjectQueryablePrimitives = Omit<Project, ProjectRelationshipList>;
// Type 3. AllPrimitives
export type ProjectAllPrimitives = ProjectQueryablePrimitives;
// type 4. Database shape
export type ProjectDB = ProjectAllPrimitives &
    Pick<Omit<Project, 'creator' | 'owner'>, 'wallets' | 'reports' | 'comments' | 'routines'> &
{
    user: User;
    organization: Organization;
    createdByUser: User;
    createdByOrganization: Organization;
    resources: { resource: Resource }[],
    starredBy: { user: User }[],
    parent: { project: Project }[],
    forks: { project: Project }[],
    tags: { tag: Tag }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Custom component for creating project. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
 const projectCreater = (toDB: FormatConverter<Project, ProjectDB>['toDB'], prisma?: PrismaType) => ({
    async create(
        data: any, 
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<ProjectDB> | null> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Remove any relationships should not be created/connected in this operation
        data = keepOnly(data, ['resources', 'parent', 'tags']);
        // Perform additional checks
        // TODO
        // Create
        const { id } = await prisma.project.create({ data });
        // Query database
        return await prisma.user.findUnique({ where: { id }, ...selectHelper<Project, ProjectDB>(info, toDB) }) as RecursivePartial<ProjectDB> | null;
    }
})

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<Project, ProjectDB> => {
    const joinMapper = {
        resources: 'resource',
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
        starredBy: 'user',
        forks: 'fork',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Project>): RecursivePartial<ProjectDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<ProjectDB>): RecursivePartial<Project> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            modified = addCreatorField(modified);
            modified = addOwnerField(modified);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<ProjectSortBy> => ({
    defaultSort: ProjectSortBy.DateUpdatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ProjectSortBy.AlphabeticalAsc]: { name: 'asc' },
            [ProjectSortBy.AlphabeticalDesc]: { name: 'desc' },
            [ProjectSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [ProjectSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [ProjectSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [ProjectSortBy.ForksDesc]: { forks: { _count: 'desc' } },
            [ProjectSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ProjectSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ProjectSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ProjectSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ProjectSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [ProjectSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
            [ProjectSortBy.VotesAsc]: { votes: { _count: 'asc' } },
            [ProjectSortBy.VotesDesc]: { votes: { _count: 'desc' } },
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
 export const projectSearcher = (
    toDB: FormatConverter<Project, ProjectDB>['toDB'],
    toGraphQL: FormatConverter<Project, ProjectDB>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    async search(where: { [x: string]: any }, input: ProjectSearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        const userIdQuery = input.userId ? { projects: { some: { userId: input.userId } } } : {};
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        const parentIdQuery = input.parentId ? { forks: { some: { forkId: input.parentId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const search = searcher<ProjectSortBy, ProjectSearchInput, Project, ProjectDB>(MODEL_TYPES.Project, toDB, toGraphQL, sorter, prisma);
        return search.search({...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where}, input, info);
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ProjectModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Project;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<ProjectCountInput>(model, prisma),
        ...creater<ProjectInput, Project, ProjectDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Project, ProjectDB>(model, format.toDB, prisma),
        ...projectCreater(format.toDB, prisma),
        ...projectSearcher(format.toDB, format.toGraphQL, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================