import { CODE } from "@local/shared";
import { CustomError } from "../error";
import { GraphQLResolveInfo } from "graphql";
import { PrismaType, RecursivePartial } from "types";
import { FindByIdInput, Organization, Routine, Standard, StandardCountInput, StandardInput, StandardSearchInput, StandardSortBy, Tag, User } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, counter, creater, deleter, findByIder, FormatConverter, InfoType, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeCreatorField, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type StandardRelationshipList = 'tags' | 'routineInputs' | 'routineOutputs' | 'starredBy' |
    'reports' | 'comments';
// Type 2. QueryablePrimitives
export type StandardQueryablePrimitives = Omit<Standard, StandardRelationshipList>;
// Type 3. AllPrimitives
export type StandardAllPrimitives = StandardQueryablePrimitives;
// type 4. Database shape
export type StandardDB = StandardAllPrimitives &
    Pick<Omit<Standard, 'creator'>, 'reports' | 'comments'> &
{
    user: User;
    organization: Organization;
    tags: { tag: Tag }[],
    routineInputs: { routine: Routine }[],
    routineOutputs: { routine: Routine }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Custom component for creating standard. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
const standardCreater = (toDB: FormatConverter<Standard, StandardDB>['toDB'], prisma?: PrismaType) => ({
    async create(
        data: any,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<StandardDB> | null> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Remove any relationships should not be created/connected in this operation
        data = keepOnly(data, ['user', 'organization', 'tags']);
        // Perform additional checks
        // TODO
        // Create
        const { id } = await prisma.standard.create({ data });
        // Query database
        return await prisma.standard.findUnique({ where: { id }, ...selectHelper<Standard, StandardDB>(info, toDB) }) as RecursivePartial<StandardDB> | null;
    }
})

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Standard, StandardDB> => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Standard>): RecursivePartial<StandardDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            // Remove isUpvoted, as it is calculated in its own query
            if (modified.isUpvoted) delete modified.isUpvoted;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<StandardDB>): RecursivePartial<Standard> => {
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
    defaultSort: StandardSortBy.AlphabeticalDesc,
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
const standarder = (format: FormatConverter<Standard, StandardDB>, sort: Sortable<StandardSortBy>, prisma?: PrismaType) => ({
    async findStandard(
        userId: string | null,
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper<Standard, StandardDB>(info, formatter().toDB);
        // Access database
        let standard = await prisma.standard.findUnique({ where: { id: input.id }, ...select });
        // Return standard with "isUpvoted" field. This must be queried separately.
        if (!userId || !standard) return standard;
        const vote = await prisma.vote.findFirst({ where: { userId, standardId: standard.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        return { ...standard, isUpvoted };
    },
    async searchStandards(
        where: { [x: string]: any },
        userId: string | null,
        input: StandardSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
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
        const search = searcher<StandardSortBy, StandardSearchInput, Standard, StandardDB>(MODEL_TYPES.Standard, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...reportIdQuery, ...routineIdQuery, ...where }, input, info);
        // Compute "isUpvoted" field for each standard
        // If userId not provided, then "isUpvoted" is null
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isUpvotedArray = await prisma.vote.findMany({ where: { userId, standardId: { in: resultIds } } });
        console.log('isUpvotedArray', isUpvotedArray)
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            console.log('ids', node.id, isUpvotedArray.map(({ standardId }) => standardId));
            const isUpvoted = isUpvotedArray.find(({ standardId }) => standardId === node.id)?.isUpvote ?? null;
            return { cursor, node: { ...node, isUpvoted } };
        });
        return searchResults;
    },
    async addStandard(
        userId: string,
        input: StandardInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        if (!prisma || !input.name || input.name.length < 1) throw new CustomError(CODE.InvalidArgs);
        // Check for censored words
        if (hasProfanity(input.name) || hasProfanity(input.description ?? '')) throw new CustomError(CODE.BannedWord);
        // Create standard data
        let standardData: { [x: string]: any } = {
            name: input.name,
            description: input.description,
            default: input.default,
            isFile: input.isFile,
            schema: input.schema,
            type: input.type
        };
        // Associate with either organization or user
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(input.organizationId, userId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            standardData = { 
                ...standardData, 
                createdByOrganization: { connect: { id: input.organizationId } },
            };
        } else {
            standardData = { 
                ...standardData, 
                createdByUser: { connect: { id: userId } },
            };
        }
        // TODO tags
        // Create standard
        const standard = await prisma.standard.create({
            data: standardData as any,
            ...selectHelper<Standard, StandardDB>(info, format.toDB)
        })
        // Return standard with "isUpvoted" field. Will be false in this case.
        return { ...standard, isUpvoted: false };
    },
    async deleteStandard(userId: string, input: any): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Find standard
        const standard = await prisma.standard.findFirst({
            where: {
                OR: [
                    { createdByOrganizationId: input.organizationId },
                    { createdByUserId: userId },
                ]
            }
        })
        if (!standard) throw new CustomError(CODE.ErrorUnknown);
        // Make sure the user is an admin of the standard
        const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(input.standardId, userId);
        if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        // Delete standard
        await prisma.standard.delete({
            where: { id: standard.id },
        });
        return true;
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StandardModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Standard;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<StandardCountInput>(model, prisma),
        ...standardCreater(format.toDB, prisma),
        ...standarder(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================