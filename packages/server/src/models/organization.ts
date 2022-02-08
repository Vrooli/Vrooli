import { PrismaType, RecursivePartial } from "../types";
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationAddInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Success } from "../schema/types";
import { addCountQueries, addJoinTables, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { GraphQLResolveInfo } from "graphql";
import { CustomError } from "../error";
import { CODE, MemberRole, organizationAdd, organizationUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { ResourceModel } from "./resource";
import { organization } from "@prisma/client";
import { TagModel } from "./tag";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Handles the authorized adding, updating, and deleting of organizations.
 */
 const organizationer = (format: FormatConverter<Organization, organization>, sort: Sortable<OrganizationSortBy>, prisma: PrismaType) => ({
    async findOrganization(
        userId: string | null | undefined, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Organization> | null> {
        // Create selector
        const select = selectHelper<Organization, organization>(info, format.toDB);
        // Access database
        let organization = await prisma.organization.findUnique({ where: { id: input.id }, ...select });
        // Return organization with "isStarred" field. This must be queried separately.
        if (!organization) throw new CustomError(CODE.InternalError, 'Organization not found');
        if (!userId) return { ...format.toGraphQL(organization), isStarred: false };
        const star = await prisma.star.findFirst({ where: { byId: userId, organizationId: organization.id } });
        const isStarred = Boolean(star) ?? false;
        const isAdmin = await this.isOwnerOrAdmin(userId, organization.id);
        return { ...format.toGraphQL(organization), isStarred, isAdmin };
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
        const search = searcher<OrganizationSortBy, OrganizationSearchInput, Organization, organization>(MODEL_TYPES.Organization, format.toDB, format.toGraphQL, sort, prisma);
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
    ): Promise<RecursivePartial<Organization>> {
        // Check for valid arguments
        organizationAdd.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = { 
            name: input.name, 
            bio: input.bio ,
            // Add user as member
            members: { create: { role: MemberRole.Owner, user: { connect: { id: userId }} } },
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, true),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, true),
        };
        // Create organization
        const organization = await prisma.organization.create({
            data: organizationData,
            ...selectHelper<Organization, organization>(info, format.toDB)
        })
        console.log('organization here', organization);
        console.log('tags', (organization as any).tags[0]);
        // Return organization with "isStarred" field. This will be its default values.
        return { ...format.toGraphQL(organization), isStarred: false, isAdmin: true };
    },
    async updateOrganization(
        userId: string,
        input: OrganizationUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Organization>> {
        // Check for valid arguments
        organizationUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = { 
            name: input.name, 
            bio: input.bio,
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, false),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, false),
        };
        // Check if user is allowed to update
        const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.id);
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        // Handle members TODO
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
            ...selectHelper<Organization, organization>(info, format.toDB)
        });
        // Return organization with "isStarred" field. This must be queried separately.
        const star = await prisma.star.findFirst({ where: { byId: userId, organizationId: organization.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(organization), isStarred, isAdmin: true };
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
 * Component for formatting between graphql and prisma types
 */
 export const organizationFormatter = (): FormatConverter<Organization, organization> => {
    const joinMapper = {
        projects: 'project',
        starredBy: 'user',
        tags: 'tag',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Organization>): RecursivePartial<organization> => {
            console.log('organization todb', obj);
            let modified = addJoinTables(obj, joinMapper);
            console.log('add join tables', modified);
            modified = addCountQueries(modified, countMapper);
            console.log('add count queries', modified);
            // Remove isStarred, as it is calculated in its own query
            if (modified.isStarred) delete modified.isStarred;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<organization>): RecursivePartial<Organization> => {
            console.log('organization tographql', obj);
            let modified = removeJoinTables(obj, joinMapper);
            console.log('removed join tables', modified);
            modified = removeCountQueries(modified, countMapper);
            console.log('removed count queries', modified);
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
    }
}

//==============================================================
/* #endregion Model */
//==============================================================