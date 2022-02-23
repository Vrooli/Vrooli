import { CODE, MemberRole, standardCreate, standardUpdate } from "@local/shared";
import { CustomError } from "../error";
import { PrismaType, RecursivePartial } from "types";
import { DeleteOneInput, FindByIdInput, Standard, StandardCountInput, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSortBy, Success } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipToPrisma, removeCountQueries, removeCreatorField, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { standard } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
export const standardFormatter = (): FormatConverter<Standard, standard> => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Standard>): RecursivePartial<standard> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            // Remove calculated fields
            delete modified.isUpvoted;
            delete modified.isStarred;
            delete modified.role;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<standard>): RecursivePartial<Standard> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            modified = addCreatorField(modified);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<StandardSortBy> => ({
    defaultSort: StandardSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [StandardSortBy.AlphabeticalAsc]: { name: 'asc' },
            [StandardSortBy.AlphabeticalDesc]: { name: 'desc' },
            [StandardSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [StandardSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [StandardSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [StandardSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [StandardSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [StandardSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [StandardSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [StandardSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
            [StandardSortBy.VotesAsc]: { score: 'asc' },
            [StandardSortBy.VotesDesc]: { score: 'desc' },
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
 * Handles the authorized adding, updating, and deleting of standards.
 */
const standarder = (format: FormatConverter<Standard, standard>, sort: Sortable<StandardSortBy>, prisma: PrismaType) => ({
    /**
     * Add, update, or remove a routine relationship from a routine list
     */
    relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        console.log('in standarder.relationshipBuilder');
        const fieldExcludes: string[] = [];
        // Convert input to Prisma shape, excluding nodes and auth fields
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'standard', isAdd, fieldExcludes })
        const tagModel = TagModel(prisma);
        console.log('standarder.relationshipBuilder formattedInput', formattedInput);
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                standardCreate.omit(fieldExcludes).validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as StandardCreateInput);
                // Handle tags
                data.tags = tagModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                standardUpdate.omit(fieldExcludes).validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as StandardUpdateInput)
                // Handle tags
                data.tags = tagModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async find(
        userId: string | null | undefined, // Of the user making the request, not the standard
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Standard> | null> {
        // Create selector
        const select = selectHelper<Standard, standard>(info, standardFormatter().toDB);
        // Access database
        let standard = await prisma.standard.findUnique({ where: { id: input.id }, ...select });
        // Return standard with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!standard) throw new CustomError(CODE.InternalError, 'Standard not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(standard)], {});
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: StandardSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : {};
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const routineIdQuery = input.routineId ? {
            OR: [
                { routineInputs: { some: { routineId: input.routineId } } },
                { routineOutputs: { some: { routineId: input.routineId } } },
            ]
        } : {};
        // Search
        const search = searcher<StandardSortBy, StandardSearchInput, Standard, standard>(MODEL_TYPES.Standard, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...reportIdQuery, ...routineIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, {});
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: StandardCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Standard>> {
        // Check for valid arguments
        standardCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create standard data
        let standardData: { [x: string]: any } = {
            name: input.name,
            description: input.description,
            default: input.default,
            isFile: input.isFile,
            schema: input.schema,
            type: input.type,
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, true),
        };
        // Associate with either organization or user
        if (input.createdByOrganizationId) {
            // Make sure the user is an admin of the organization
            const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            standardData = {
                ...standardData,
                createdByOrganization: { connect: { id: input.createdByOrganizationId } },
            };
        } else {
            standardData = {
                ...standardData,
                createdByUser: { connect: { id: userId } },
            };
        }
        // Create standard
        const standard = await prisma.standard.create({
            data: standardData as any,
            ...selectHelper<Standard, standard>(info, format.toDB)
        })
        // Return project with "role", "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.toGraphQL(standard), role: MemberRole.Owner as any, isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: StandardUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Standard>> {
        // Check for valid arguments
        standardUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create standard data
        let standardData: { [x: string]: any } = {
            description: input.description,
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, false),
        };
        // Check if authorized to update
        let standard = await prisma.standard.findUnique({
            where: { id: input.id },
            select: {
                id: true,
                createdByUserId: true,
                createdByOrganizationId: true,
            }
        });
        if (!standard) throw new CustomError(CODE.NotFound, 'Standard not found');
        if (standard.createdByUserId && standard.createdByUserId !== userId) throw new CustomError(CODE.Unauthorized);
        if (standard.createdByOrganizationId) {
            const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, standard.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        }
        // Update standard
        standard = await prisma.standard.update({
            where: { id: standard.id },
            data: standardData as any,
            ...selectHelper<Standard, standard>(info, format.toDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(standard)], {});
        return formatted[0];
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const standard = await prisma.standard.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                createdByUserId: true,
                createdByOrganizationId: true,
            }
        })
        if (!standard) throw new CustomError(CODE.NotFound, "Standard not found");
        // Check if user is authorized
        let authorized = userId === standard.createdByUserId;
        if (!authorized && standard.createdByOrganizationId) {
            [authorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, standard.createdByOrganizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.standard.delete({
            where: { id: standard.id },
        });
        return { success: true };
    },
    /**
     * Supplemental fields are role, isUpvoted, and isStarred
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<Standard>[],
        known: { [x: string]: any[] }, // Known values (i.e. don't need to query), in same order as objects
    ): Promise<RecursivePartial<Standard>[]> {
        // If userId not provided, return the input with isStarred false, isUpvoted null, and role null
        if (!userId) return objects.map(x => ({ ...x, isStarred: false, isUpvoted: null, role: null }));
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Check if isStarred is provided
        if (known.isStarred) objects = objects.map((x, i) => ({ ...x, isStarred: known.isStarred[i] }));
        // Otherwise, query for isStarred
        else {
            const isStarredArray = await StarModel(prisma).getIsStarreds(userId, ids, 'standard');
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Check if isUpvoted is provided
        if (known.isUpvoted) objects = objects.map((x, i) => ({ ...x, isUpvoted: known.isUpvoted[i] }));
        // Otherwise, query for isStarred
        else {
            const isUpvotedArray = await VoteModel(prisma).getIsUpvoteds(userId, ids, 'standard');
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Check is role is provided
        if (known.role) objects = objects.map((x, i) => ({ ...x, role: known.role[i] }));
        // Otherwise, query for role
        else {
            console.log('standard supplemental fields', objects)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => x.creator?.__typename === 'Organization')
                .map(x => x.id)
                .filter(x => Boolean(x)) as string[];
            const roles = await OrganizationModel(prisma).getRoles(userId, organizationIds);
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: x.creator?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        return objects;
    },
    profanityCheck(data: StandardCreateInput | StandardUpdateInput): void {
        if (hasProfanity((data as any)?.name ?? '', data.description)) throw new CustomError(CODE.BannedWord);
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StandardModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Standard;
    const format = standardFormatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<StandardCountInput>(model, prisma),
        ...standarder(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================