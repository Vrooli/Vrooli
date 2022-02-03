import { PrismaType, RecursivePartial } from "../types";
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationAddInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Project, Resource, Success, Tag, User } from "../schema/types";
import { addCountQueries, addJoinTables, counter, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { GraphQLResolveInfo } from "graphql";
import { CustomError } from "../error";
import { CODE, MemberRole, organizationAdd, organizationUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { ResourceModel } from "./resource";

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
        userId: string | null | undefined, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Create selector
        const select = selectHelper<Organization, OrganizationDB>(info, format.toDB);
        // Access database
        let organization = await prisma.organization.findUnique({ where: { id: input.id }, ...select });
        // Return organization with "isStarred" field. This must be queried separately.
        if (!organization) throw new CustomError(CODE.InternalError, 'Organization not found');
        if (!userId) return { ...organization, isStarred: false };
        const star = await prisma.star.findFirst({ where: { byId: userId, organizationId: organization.id } });
        const isStarred = Boolean(star) ?? false;
        const isAdmin = await this.isOwnerOrAdmin(userId, organization.id);
        return { ...organization, isStarred, isAdmin };
    },
    async searchOrganizations(
        where: { [x: string]: any },
        userId: string | null | undefined,
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
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isStarred: false, isAdmin: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, organizationId: { in: resultIds } } });
        const isAdminArray = await prisma.organization_users.findMany({ where: { 
            organizationId: { in: resultIds },
            user: { id: userId },
            role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
        } });
        console.log('isStarredArray', isStarredArray);
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            const isStarred = Boolean(isStarredArray.find(({ organizationId }) => organizationId === node.id));   
            const isAdmin = Boolean(isAdminArray.find(({ organizationId }) => organizationId === node.id));
            return { cursor, node: { ...node, isStarred, isAdmin } };
        });
        return searchResults;
    },
    async addOrganization(
        userId: string,
        input: OrganizationAddInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        organizationAdd.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = { name: input.name, bio: input.bio };
        // Add user as member
        organizationData.members = { connect: { id: userId } };
        // Handle resources
        const resourceData = ResourceModel(prisma).relationshipBuilder(userId, input, true);
        if (resourceData) organizationData.resources = resourceData;
        // Create organization
        const organization = await prisma.organization.create({
            data: organizationData,
            ...selectHelper<Organization, OrganizationDB>(info, format.toDB)
        })
        // Return organization with "isStarred" field. This will be its default values.
        return { ...organization, isStarred: false, isAdmin: true };
    },
    async updateOrganization(
        userId: string,
        input: OrganizationUpdateInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        organizationUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = { name: input.name, bio: input.bio };
        // Check if user is allowed to update
        const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.id ?? '');
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        // Handle resources
        const resourceData = ResourceModel(prisma).relationshipBuilder(userId, input, false);
        if (resourceData) organizationData.resources = resourceData;
        // Find organization
        let organization = await prisma.organization.findFirst({
            where: {
                AND: [
                    { id: input.id ?? '' },
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
        return { ...organization, isStarred, isAdmin: true };
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
        ...organizationer(format, sort, prisma),
        ...organizationCreater(format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================