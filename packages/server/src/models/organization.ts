import { PrismaType, RecursivePartial } from "../types";
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationInput, OrganizationSearchInput, OrganizationSortBy, Project, Resource, Routine, Success, Tag, User } from "../schema/types";
import { addCountQueries, addJoinTables, counter, deleter, findByIder, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { GraphQLResolveInfo } from "graphql";
import { CustomError } from "../error";
import { CODE, MemberRole } from "@local/shared";
import { hasProfanity } from "../utils/censor";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type OrganizationRelationshipList = 'comments' | 'resources' | 'wallets' | 'projects' | 'starredBy' |
    'routines' | 'routinesCreated' | 'tags' | 'reports' | 'members';
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
    members: { user: User }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Handles the authorized adding, updating, and deleting of organizations.
 */
 const organizationer = (format: FormatConverter<Organization, OrganizationDB>, sort: Sortable<OrganizationSortBy>, prisma: PrismaType) => ({
    async findOrganization(
        userId: string | null, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Create selector
        const select = selectHelper<Organization, OrganizationDB>(info, organizationFormatter().toDB);
        // Access database
        let organization = await prisma.organization.findUnique({ where: { id: input.id }, ...select });
        // Return organization with "isStarred" field. This must be queried separately.
        if (!userId || !organization) return organization;
        const star = await prisma.star.findFirst({ where: { byId: userId, organizationId: organization.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...organization, isStarred };
    },
    async searchOrganizations(
        where: { [x: string]: any },
        userId: string | null,
        input: OrganizationSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Many-to-many search queries
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        // One-to-many search queries
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const userIdQuery = input.userId ? { members: { some: { id: input.userId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        // Search
        const search = searcher<OrganizationSortBy, OrganizationSearchInput, Organization, OrganizationDB>(MODEL_TYPES.Organization, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...projectIdQuery, ...routineIdQuery, ...userIdQuery, ...reportIdQuery, ...standardIdQuery, ...where }, input, info);
        // Compute "isStarred" fields for each organization
        // If userId not provided, then "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null, isStarred: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, organizationId: { in: resultIds } } });
        console.log('isStarredArray', isStarredArray);
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            const isStarred = Boolean(isStarredArray.find(({ organizationId }) => organizationId === node.id));   
            return { cursor, node: { ...node, isStarred } };
        });
        return searchResults;
    },
    async addOrganization(
        userId: string,
        input: OrganizationInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.name || input.name.length < 1) throw new CustomError(CODE.InternalError, 'Name too short');
        // Check for censored words
        if (hasProfanity(input.name) || hasProfanity(input.name ?? '') || hasProfanity(input.bio ?? '')) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = { name: input.name, bio: input.bio };
        // Add user as member
        organizationData.members = { connect: { id: userId } };
        // TODO resources
        // Create organization
        const organization = await prisma.organization.create({
            data: organizationData as any,
            ...selectHelper<Organization, OrganizationDB>(info, format.toDB)
        })
        // Return organization with "isStarred" field. This will be its default values.
        return { ...organization, isStarred: false };
    },
    async updateOrganization(
        userId: string,
        input: OrganizationInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.id) throw new CustomError(CODE.InternalError, 'No organization id provided');
        if (!input.name || input.name.length < 1) throw new CustomError(CODE.InternalError, 'Name too short');
        // Check for censored words
        if (hasProfanity(input.name) || hasProfanity(input.name ?? '') || hasProfanity(input.bio ?? '')) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = { name: input.name, bio: input.bio };
        // Add user as member
        organizationData.members = { connect: { id: userId } };
        // TODO resources
        // Find organization
        let organization = await prisma.organization.findFirst({
            where: {
                AND: [
                    { id: input.id },
                    { members: { some: { id: userId } } },
                ]
            }
        })
        if (!organization) throw new CustomError(CODE.ErrorUnknown);
        // Update organization
        organization = await prisma.organization.update({
            where: { id: organization.id },
            data: organizationData as any,
            ...selectHelper<Organization, OrganizationDB>(info, format.toDB)
        });
        // Return organization with "isStarred" field. This must be queried separately.
        const star = await prisma.star.findFirst({ where: { byId: userId, organizationId: organization.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...organization, isStarred };
    },
    async deleteOrganization(userId: string, input: DeleteOneInput): Promise<Success> {
        // Make sure the user is an admin of this organization
        const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.id);
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        await prisma.organization.delete({
            where: { id: input.id }
        })
        return { success: true };
    },
    async isOwnerOrAdmin (userId: string, organizationId: string): Promise<boolean> {
        const memberData = await prisma.organization_users.findFirst({ 
            where: {
                organization: { id: organizationId },
                user: { id: userId },
                role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
            }
        });
        console.log('member data', memberData);
        return Boolean(memberData);
    }
})

/**
 * Custom component for creating organization. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
 const organizationCreater = (toDB: FormatConverter<Organization, OrganizationDB>['toDB'], prisma: PrismaType) => ({
    async create(
        data: any, 
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<OrganizationDB> | null> {
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
 export const organizationFormatter = (): FormatConverter<Organization, OrganizationDB> => {
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
            // Remove isStarred, as it is calculated in its own query
            if (modified.isStarred) delete modified.isStarred;
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
export const organizationSorter = (): Sortable<OrganizationSortBy> => ({
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

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function OrganizationModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Organization;
    const format = organizationFormatter();
    const sort = organizationSorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<OrganizationCountInput>(model, prisma),
        ...findByIder<Organization, OrganizationDB>(model, format.toDB, prisma),
        ...organizationer(format, sort, prisma),
        ...organizationCreater(format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================