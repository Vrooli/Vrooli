import { PartialSelectConvert, PrismaType, RecursivePartial } from "../types";
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy, Success, Count, DeleteManyInput } from "../schema/types";
import { addJoinTables, counter, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipFormatter, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, MemberRole, organizationCreate, organizationUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { ResourceModel } from "./resource";
import { organization, organization_users } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import _ from "lodash";

export const organizationDBFields = ['id', 'created_at', 'updated_at', 'bio', 'name', 'openToNewMembers', 'stars'];

//==============================================================
/* #region Custom Components */
//==============================================================

type OrganizationFormatterType = FormatConverter<Organization, organization>;
/**
 * Component for formatting between graphql and prisma types
 */
export const organizationFormatter = (): OrganizationFormatterType => {
    const joinMapper = {
        starredBy: 'user',
        tags: 'tag',
    };
    return {
        dbShape: (partial: PartialSelectConvert<Organization>): PartialSelectConvert<organization> => {
            let modified = partial;
            console.log('organization toDB shape', modified);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbShape],
                ['members', (d: any) => ({
                    ...d,
                    user: FormatterMap.User.dbShapeUser(d.user),
                })],
                ['projects', FormatterMap.Project.dbShape],
                ['projectsCreated', FormatterMap.Project.dbShape],
                ['reports', FormatterMap.Report.dbShape],
                ['resources', FormatterMap.Resource.dbShape],
                ['routines', FormatterMap.Routine.dbShape],
                ['routinesCreated', FormatterMap.Routine.dbShape],
                ['standards', FormatterMap.Standard.dbShape],
                ['starredBy', FormatterMap.User.dbShapeUser],
                ['tags', FormatterMap.Tag.dbShape],
            ]);
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<organization> => {
            console.log('organization toDB prune', info);
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            console.log('organization aaaaaa', info)
            // Remove calculated fields
            let { isStarred, role, ...rest } = modified;
            modified = rest;
            // Convert relationships
            console.log('organization bbbbbb', info)
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbPrune],
                ['members', (d: any) => ({
                    ...d,
                    user: FormatterMap.User.dbPruneUser(d.user),
                })],
                ['projects', FormatterMap.Project.dbPrune],
                ['projectsCreated', FormatterMap.Project.dbPrune],
                ['reports', FormatterMap.Report.dbPrune],
                ['resources', FormatterMap.Resource.dbPrune],
                ['routines', FormatterMap.Routine.dbPrune],
                ['routinesCreated', FormatterMap.Routine.dbPrune],
                ['standards', FormatterMap.Standard.dbPrune],
                ['starredBy', FormatterMap.User.dbPruneUser],
                ['tags', FormatterMap.Tag.dbPrune],
            ]);
            console.log('organization toDB prune complete here is info param that should be the same', info);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<organization> => {
            return organizationFormatter().dbShape(organizationFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<organization>): RecursivePartial<Organization> => {
            console.log('organization tographql', obj);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            let modified = removeJoinTables(obj, joinMapper);
            // Convert relationships
            modified = relationshipFormatter(obj, [
                ['comments', FormatterMap.Comment.selectToGraphQL],
                ['members', (d: any) => ({
                    ...d,
                    user: FormatterMap.User.selectToGraphQLUser(d.user),
                })],
                ['projects', FormatterMap.Project.selectToGraphQL],
                ['projectsCreated', FormatterMap.Project.selectToGraphQL],
                ['reports', FormatterMap.Report.selectToGraphQL],
                ['resources', FormatterMap.Resource.selectToGraphQL],
                ['routines', FormatterMap.Routine.selectToGraphQL],
                ['routinesCreated', FormatterMap.Routine.selectToGraphQL],
                ['standards', FormatterMap.Standard.selectToGraphQL],
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
                ['tags', FormatterMap.Tag.selectToGraphQL],
            ]);
            console.log('organization tographql relationships converted', modified);
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
            console.log('organization tographql final', modified);
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
            [OrganizationSortBy.StarsAsc]: { stars: 'asc' },
            [OrganizationSortBy.StarsDesc]: { stars: 'desc' },
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

interface OrganizationCRUDInput {
    userId: string | null,
    readOne: FindByIdInput, // Non-paginated search
    readMany: OrganizationSearchInput, // Paginated search
    createMany: OrganizationCreateInput[],
    updateMany: OrganizationUpdateInput[],
    deleteMany: string[],
    info: InfoType,
}

interface OrganizationCRUDResult {
    foundOne: RecursivePartial<Organization> | null,
    foundMany: PaginatedSearchResult,
    created: RecursivePartial<Organization>[],
    updated: RecursivePartial<Organization>[],
    deleted: Count, // Number of deleted organizations
}

/**
 * Handles the authorized adding, updating, and deleting of organizations.
 */
const organizationer = (format: OrganizationFormatterType, sort: Sortable<OrganizationSortBy>, prisma: PrismaType) => ({
    async find(
        userId: string | null | undefined, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType,
    ): Promise<RecursivePartial<Organization> | null> {
        // Create selector
        const select = selectHelper(info, format.selectToDB);
        // Access database
        let organization = await prisma.organization.findUnique({ where: { id: input.id }, ...select });
        // Check if exists
        if (!organization) throw new CustomError(CODE.InternalError, 'Organization not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [organization], info);
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: OrganizationSearchInput,
        info: InfoType,
    ): Promise<PaginatedSearchResult> {
        console.log('search organizations', info)
        // Many-to-many search queries
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        // One-to-many search queries
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const userIdQuery = input.userId ? { members: { some: { id: input.userId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        // Search
        const search = searcher<OrganizationSortBy, OrganizationSearchInput, Organization, organization>(MODEL_TYPES.Organization, format.selectToDB, sort, prisma);
        console.log('calling organization search...', info)
        let searchResults = await search.search({ ...projectIdQuery, ...routineIdQuery, ...userIdQuery, ...reportIdQuery, ...standardIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        console.log('organization searchResults BEFORE supplemental fields', info);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, info);
        console.log('organization searchResults AFTER supplemental fields', formattedNodes);
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: OrganizationCreateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Organization>> {
        // Check for valid arguments
        organizationCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
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
            ...selectHelper(info, format.selectToDB)
        })
        console.log('organization here', organization);
        console.log('tags', (organization as any).tags[0]);
        // Return organization with "isStarred" field. This will be its default values.
        return { ...format.selectToGraphQL(organization), isStarred: false, role: MemberRole.Owner as any };
    },
    async update(
        userId: string,
        input: OrganizationUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Organization>> {
        // Check for valid arguments
        organizationUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
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
            ...selectHelper(info, format.selectToDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [organization], info);
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
     * Combines all crud operations to save on database calls
     */
    async crud({ userId, readOne, readMany, createMany, updateMany, deleteMany }: OrganizationCRUDInput): Promise<OrganizationCRUDResult> {
        /**
         * Helper function for creating create/update Prisma value
         */
        const createData = async (input: OrganizationCreateInput | OrganizationUpdateInput): Promise<{ [x: string]: any }> => ({
            name: input.name,
            bio: input.bio,
            // Handle resources
            resources: ResourceModel(prisma).relationshipBuilder(userId, input, false),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, false),
        })
        // Validate inputs
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => organizationCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => this.profanityCheck(input));
            // Check if user will pass max organizations limit
            const existingCount = await prisma.organization.count({ where: { members: { some: { userId: userId ?? '', role: MemberRole.Owner as any } } } });
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.ErrorUnknown, 'To prevent spam, user cannot own more than 100 organizations. If you think this is a mistake, please contact us');
            }
        }
        if (updateMany) {
            updateMany.forEach(input => organizationUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => this.profanityCheck(input));
        }
        if (deleteMany) {
            // Check that user is owner of each organization
            const roles = userId
                ? await this.getRoles(userId, deleteMany)
                : Array(deleteMany.length).fill(null);
            if (roles.some(role => role !== MemberRole.Owner)) throw new CustomError(CODE.Unauthorized);
        }
        // Perform mutations
        let foundOne: any, foundMany: any, created: any, updated: any, deleted: any;
        //TODO
        // Perform queries
        //TODO
        // Format and add supplemental/calculated fields
        //TODO
        return { foundOne, foundMany, created, updated, deleted };
    },
    /**
     * Supplemental fields are isOwn and role
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Organization>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
        console.log('organization supplementalfields', objects)
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            console.log('organization isStarred query');
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'organization')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for role
        if (partial.role) {
            const roles = userId
                ? await this.getRoles(userId, ids)
                : Array(ids.length).fill(null);
            objects = objects.map((x, i) => ({ ...x, role: roles[i] })) as any;
        }
        // Perform nested supplemental fields
        if (partial.tags) {
            console.log('in organization partial.tags true', partial.tags)
            // Find tags in each object, accounting for join table
            let tags = objects.map(x => Array.isArray(x.tags) ? x.tags.map(t => t.tag) : []).reduce((a, b) => a.concat(b), []);
            console.log('found tags', tags);
            // Query for tag supplemental fields
            tags = await TagModel(prisma).supplementalFields(userId, tags, partial.tags);
            console.log('organization tags supplemental fields', tags)
            // Update each tag object in objects
            let tagIndex = 0;
            for (let i = 0; i < objects.length; i++) {
                const numCurrTags = objects[i].tags?.length ?? 0;
                if (tagIndex + numCurrTags > tags.length) break;
                objects[i].tags = tags.slice(tagIndex, tagIndex + numCurrTags);
                tagIndex += numCurrTags;
            }
        }
        console.log('organization going to convert to graphql', objects);
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
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
    },
    profanityCheck(data: OrganizationCreateInput | OrganizationUpdateInput): void {
        if (hasProfanity(data.name, data.bio)) throw new CustomError(CODE.BannedWord);
    },
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