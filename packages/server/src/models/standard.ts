import { CODE, MemberRole, standardCreate, standardUpdate } from "@local/shared";
import { CustomError } from "../error";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { DeleteOneInput, FindByIdInput, Standard, StandardCountInput, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSortBy, Success } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, counter, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipFormatter, relationshipToPrisma, removeCountQueries, removeCreatorField, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { standard } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import _ from "lodash";

export const standardDBFields = ['id', 'created_at', 'updated_at', 'default', 'description', 'isFile', 'name', 'schema', 'type', 'version', 'createdByUserId', 'createdByOrganizationId', 'stars', 'score']

//==============================================================
/* #region Custom Components */
//==============================================================

type StandardFormatterType = FormatConverter<Standard, standard>;
/**
 * Component for formatting between graphql and prisma types
 */
export const standardFormatter = (): StandardFormatterType => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        dbShape: (partial: PartialSelectConvert<Standard>): PartialSelectConvert<standard> => {
            let modified = partial;
            // Deconstruct GraphQL unions
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbShape],
                ['reports', FormatterMap.Report.dbShape],
                ['resources', FormatterMap.Resource.dbShape],
                ['routineInputs', (d: any) => ({
                    ...d,
                    routine: FormatterMap.Routine.dbShape(d.routine),
                })],
                ['routineOutputs', (d: any) => ({
                    ...d,
                    routine: FormatterMap.Routine.dbShape(d.routine),
                })],
                ['routines', FormatterMap.Routine.dbShape],
                ['starredBy', FormatterMap.User.dbShapeUser],
                ['tags', FormatterMap.Tag.dbShape],
            ]);
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<standard> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Remove calculated fields
            let { isUpvoted, isStarred, role, ...rest } = modified;
            modified = rest;
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbPrune],
                ['reports', FormatterMap.Report.dbPrune],
                ['resources', FormatterMap.Resource.dbPrune],
                ['routineInputs', (d: any) => ({
                    ...d,
                    routine: FormatterMap.Routine.dbPrune(d.routine),
                })],
                ['routineOutputs', (d: any) => ({
                    ...d,
                    routine: FormatterMap.Routine.dbPrune(d.routine),
                })],
                ['routines', FormatterMap.Routine.dbPrune],
                ['starredBy', FormatterMap.User.dbPruneUser],
                ['tags', FormatterMap.Tag.dbPrune],
            ]);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<standard> => {
            return standardFormatter().dbShape(standardFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<standard>): RecursivePartial<Standard> => {
            if (!_.isObject(obj)) return obj;
            // Create unions
            let modified = removeCountQueries(obj, countMapper);
            modified = addCreatorField(modified);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            modified = removeJoinTables(modified, joinMapper);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.selectToGraphQL],
                ['reports', FormatterMap.Report.selectToGraphQL],
                ['resources', FormatterMap.Resource.selectToGraphQL],
                ['routineInputs', (d: any) => ({
                    ...d,
                    routine: FormatterMap.Routine.selectToGraphQL(d.routine),
                })],
                ['routineOutputs', (d: any) => ({
                    ...d,
                    routine: FormatterMap.Routine.selectToGraphQL(d.routine),
                })],
                ['routines', FormatterMap.Routine.selectToGraphQL],
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
                ['tags', FormatterMap.Tag.selectToGraphQL],
            ]);
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
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
            [StandardSortBy.StarsAsc]: { stars: 'asc' },
            [StandardSortBy.StarsDesc]: { stars: 'desc' },
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
const standarder = (format: StandardFormatterType, sort: Sortable<StandardSortBy>, prisma: PrismaType) => ({
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
        info: InfoType,
    ): Promise<RecursivePartial<Standard> | null> {
        // Create selector
        const select = selectHelper(info, format.selectToDB);
        // Access database
        let standard = await prisma.standard.findUnique({ where: { id: input.id }, ...select });
        // Return standard with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!standard) throw new CustomError(CODE.InternalError, 'Standard not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [standard], info);
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: StandardSearchInput,
        info: InfoType,
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
        const search = searcher<StandardSortBy, StandardSearchInput, Standard, standard>(MODEL_TYPES.Standard, format.selectToDB, sort, prisma);
        console.log('calling standard search...')
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...reportIdQuery, ...routineIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, info);
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: StandardCreateInput,
        info: InfoType,
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
            ...selectHelper(info, format.selectToDB)
        })
        // Return project with "role", "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.selectToGraphQL(standard), role: MemberRole.Owner as any, isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: StandardUpdateInput,
        info: InfoType,
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
            ...selectHelper(info, format.selectToDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [standard], info);
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
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Standard>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'standard')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, 'standard')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for role
        if (partial.role) {
            console.log('standard supplemental fields', objects)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => x.creator?.__typename === 'Organization')
                .map(x => x.id)
                .filter(x => Boolean(x)) as string[];
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: x.creator?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
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