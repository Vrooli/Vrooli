import { Count, DeleteManyInput, FindByIdInput, Organization, Project, Routine, Standard, Tag, TagCountInput, TagAddInput, TagUpdateInput, TagSearchInput, TagSortBy, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addJoinTables, counter, findByIder, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, tagAdd, tagUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type TagRelationshipList = 'organizations' | 'projects' | 'routines' | 'standards' |
    'starredBy';
// Type 2. QueryablePrimitives
export type TagQueryablePrimitives = Omit<Tag, TagRelationshipList>;
// Type 3. AllPrimitives
export type TagAllPrimitives = TagQueryablePrimitives;
// type 4. Database shape
export type TagDB = TagAllPrimitives &
{
    organizations: { tagged: Organization },
    projects: { tagged: Project },
    routines: { tagged: Routine },
    standards: { tagged: Standard },
    starredBy: { user: User },
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
 export const tagFormatter = (): FormatConverter<Tag, TagDB> => {
    const joinMapper = {
        organizations: 'tagged',
        projects: 'tagged',
        routines: 'tagged',
        standards: 'tagged',
        starredBy: 'user',
    };
    return {
        toDB: (obj: RecursivePartial<Tag>): RecursivePartial<TagDB> => {
            let modified = addJoinTables(obj, joinMapper)
            // Remove isStarred, as it is calculated in its own query
            if (modified.isStarred) delete modified.isStarred;
            return modified
        },
        toGraphQL: (obj: RecursivePartial<TagDB>): RecursivePartial<Tag> => removeJoinTables(obj, joinMapper)
    }
}

/**
 * Component for search filters
 */
 export const tagSorter = (): Sortable<TagSortBy> => ({
    defaultSort: TagSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [TagSortBy.AlphabeticalAsc]: { name: 'asc' },
            [TagSortBy.AlphabeticalDesc]: { name: 'desc' },
            [TagSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [TagSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [TagSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [TagSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [TagSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [TagSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { tag: { ...insensitive } },
                { description: { ...insensitive } },
            ]
        })
    }
})

/**
 * Handles the authorized adding, updating, and deleting of tags.
 */
 const tagger = (format: FormatConverter<Tag, TagDB>, sort: Sortable<TagSortBy>, prisma: PrismaType) => ({
    async findTag(
        userId: string | null | undefined, // Of the user making the request, not the tag's creator
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<any> {
        // Create selector
        const select = selectHelper<Tag, TagDB>(info, format.toDB);
        // Access database
        let tag = await prisma.tag.findUnique({ where: { id: input.id }, ...select });
        // Return tag with "isStarred" field. This must be queried separately.
        if (!tag) throw new CustomError(CODE.InternalError, 'Tag not found');
        if (!userId) return { ...tag, isStarred: false };
        const star = await prisma.star.findFirst({ where: { byId: userId, tagId: tag.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...tag, isStarred };
    },
    async searchTags(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: TagSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // If myId or hidden specified, limit results.
        let idsLimit: string[] | undefined = undefined;
        // Looking for tags the requesting user has starred
        if (userId && input.myTags){
            idsLimit = (await prisma.star.findMany({
                where: {
                    AND: [
                        { byId: userId },
                        { NOT: { tagId: null } }
                    ]
                }
            })).map(s => s.tagId).filter(s => s !== null) as string[];
        }
        // Looking for tags the requesting user has hidden
        else if (userId && input.hidden) {
            idsLimit = (await prisma.user_tag_hidden.findMany({
                where: { userId }
            })).map(s => s.tagId).filter(s => s !== null) as string[]
        }
        // Search
        const search = searcher<TagSortBy, TagSearchInput, Tag, TagDB>(MODEL_TYPES.Tag, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...where }, {...input, ids: idsLimit}, info);
        // Compute "isStarred" field for each tag
        // If userId not provided "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isStarred: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, tagId: { in: resultIds } } });
        console.log('isStarredArray', isStarredArray);
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            const isStarred = Boolean(isStarredArray.find(({ tagId }) => tagId === node.id));   
            console.log('isStarred', isStarred, );
            return { cursor, node: { ...node, isStarred } };
        });
        return searchResults;
    },
    async addTag(
        userId: string,
        input: TagAddInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        tagAdd.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.tag, input.description)) throw new CustomError(CODE.BannedWord);
        // Create tag data
        let tagData: { [x: string]: any } = { name: input.tag, description: input.description, createdByUserId: userId };
        // Create tag
        const tag = await prisma.tag.create({
            data: tagData,
            ...selectHelper<Tag, TagDB>(info, format.toDB)
        })
        // Return tag with "isStarred" field. This will be its default value.
        return { ...tag, isStarred: false };
    },
    async updateTag(
        userId: string,
        input: TagUpdateInput,
        info: InfoType = null,
    ): Promise<any> {
        // Check for valid arguments
        tagUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.tag, input.description)) throw new CustomError(CODE.BannedWord);
        // Create tag data
        let tagData: { [x: string]: any } = { name: input.tag, description: input.description };
        // Update tag
        const tag = await prisma.tag.update({
            where: { createdByUserId: userId },
            data: tagData,
            ...selectHelper<Tag, TagDB>(info, format.toDB)
        });
        // Return tag with "isStarred" field. This must be queried separately.
        const star = await prisma.star.findFirst({ where: { byId: userId, tagId: tag.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...tag, isStarred };
    },
    async deleteTags(userId: string, input: DeleteManyInput): Promise<Count> {
        // Find
        const tags = await prisma.tag.findMany({
            where: { id: { in: input.ids } },
        })
        if (!tags || tags.length === 0) throw new CustomError(CODE.NotFound, "Tags not found");
        if (tags.some(t => t.createdByUserId !== userId)) throw new CustomError(CODE.Unauthorized, "You can't delete tags you didn't create");
        // Delete
        return await prisma.tag.deleteMany({
            where: { id: { in: tags.map(t => t.id) } },
        });
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TagModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Tag;
    const format = tagFormatter();
    const sort = tagSorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<TagCountInput>(model, prisma),
        ...findByIder<Tag, TagDB>(model, format.toDB, prisma),
        ...tagger(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================