import { PrismaType, RecursivePartial } from "../types";
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Success } from "../schema/types";
import { addCountQueries, addJoinTables, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, MemberRole, organizationCreate, organizationUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { ResourceModel } from "./resource";
import { organization, organization_users } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Handles the authorized adding, updating, and deleting of organizations.
 */
const organizationer = (format: FormatConverter<Organization, organization>, sort: Sortable<OrganizationSortBy>, prisma: PrismaType) => ({
    async find(
        userId: string | null | undefined, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Organization> | null> {
        // Create selector
        const select = selectHelper<Organization, organization>(info, format.toDB);
        // Access database
        let organization = await prisma.organization.findUnique({ where: { id: input.id }, ...select });
        // Check if exists
        if (!organization) throw new CustomError(CODE.InternalError, 'Organization not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(organization)], {});
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: OrganizationSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        console.log('search organizations', input)
        // Many-to-many search queries
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        // One-to-many search queries
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const userIdQuery = input.userId ? { members: { some: { id: input.userId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        console.log('user query', userIdQuery)
        // Search
        const search = searcher<OrganizationSortBy, OrganizationSearchInput, Organization, organization>(MODEL_TYPES.Organization, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...projectIdQuery, ...routineIdQuery, ...userIdQuery, ...reportIdQuery, ...standardIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, {});
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: OrganizationCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Organization>> {
        // Check for valid arguments
        organizationCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create organization data
        let organizationData: { [x: string]: any } = {
            name: input.name,
            bio: input.bio,
            // Add user as member
            members: { create: { role: MemberRole.Owner, user: { connect: { id: userId } } } },
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
        return { ...format.toGraphQL(organization), isStarred: false, role: MemberRole.Owner as any };
    },
    async update(
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
        const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.id);
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
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(organization)], {});
        return formatted[0];
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Make sure the user is an admin of this organization
        const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.id);
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        await prisma.organization.delete({
            where: { id: input.id }
        })
        return { success: true };
    },
    /**
     * Supplemental fields are isOwn and role
     */
     async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<Organization>[],
        known: { [x: string]: any[] }, // Known values (i.e. don't need to query), in same order as objects
    ): Promise<RecursivePartial<Organization>[]> {
        // If userId not provided, return the input with isStarred false and role null
        if (!userId) return objects.map(x => ({ ...x, isStarred: false, role: null }));
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Check if isStarred is provided
        if (known.isStarred) objects = objects.map((x, i) => ({ ...x, isStarred: known.isStarred[i] }));
        // Otherwise, query for isStarred
        else {
            const isStarredArray = await StarModel(prisma).getIsStarreds(userId, ids, 'organization');
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Check is role is provided
        if (known.role) objects = objects.map((x, i) => ({ ...x, role: known.role[i] }));
        // Otherwise, query for role
        else {
            const roles = await this.getRoles(userId, ids);
            objects = objects.map((x, i) => ({ ...x, role: roles[i] })) as any;
        }
        return objects;
    },
    async getRoles(userId: string, ids: string[]): Promise<Array<MemberRole | undefined>> {
        // Query member data for each ID
        const roleArray = await prisma.organization_users.findMany({
            where: { organizationId: { in: ids }, user: { id: userId } },
            select: { organizationId: true, role: true }
        });
        return ids.map(id => {
            const role = roleArray.find(({ organizationId }) => organizationId === id);
            return role?.role as MemberRole | undefined;
        });
    },
    async isOwnerOrAdmin(userId: string, organizationId: string): Promise<[boolean, organization_users | null]> {
        const memberData = await prisma.organization_users.findFirst({
            where: {
                organization: { id: organizationId },
                user: { id: userId },
                role: { in: [MemberRole.Admin as any, MemberRole.Owner as any] },
            }
        });
        return [Boolean(memberData), memberData];
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
            // Remove calculated fields
            delete modified.isStarred;
            delete modified.role;
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
    defaultSort: OrganizationSortBy.AlphabeticalAsc,
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