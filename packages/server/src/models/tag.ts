import { Count, DeleteManyInput, FindByIdInput, Tag, TagCountInput, TagCreateInput, TagUpdateInput, TagSearchInput, TagSortBy } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addJoinTables, counter, FormatConverter, InfoType, joinRelationshipToPrisma, MODEL_TYPES, PaginatedSearchResult, relationshipToPrisma, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, tagCreate, tagUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { tag } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
export const tagFormatter = (): FormatConverter<Tag, tag> => {
    const joinMapper = {
        organizations: 'tagged',
        projects: 'tagged',
        routines: 'tagged',
        standards: 'tagged',
        starredBy: 'user',
    };
    return {
        toDB: (obj: RecursivePartial<Tag>): RecursivePartial<tag> => {
            let modified = addJoinTables(obj, joinMapper)
            // Remove isStarred, as it is calculated in its own query
            if (modified.isStarred) delete modified.isStarred;
            return modified
        },
        toGraphQL: (obj: RecursivePartial<tag>): RecursivePartial<Tag> => removeJoinTables(obj, joinMapper)
    }
}

/**
 * Component for search filters
 */
export const tagSorter = (): Sortable<TagSortBy> => ({
    defaultSort: TagSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [TagSortBy.AlphabeticalAsc]: { tag: 'asc' },
            [TagSortBy.AlphabeticalDesc]: { tag: 'desc' },
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
const tagger = (format: FormatConverter<Tag, tag>, sort: Sortable<TagSortBy>, prisma: PrismaType) => ({
    /**
     * Add, update, or remove a tag relationship from an object. 
     * This is different than most relationship builders, as tags are a many-to-many relationship.
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        console.log('tag relationshipBuilder', input, isAdd);
        // If any tag creates, check if they're supposed to be connects
        if (Array.isArray(input.tagCreate)) {
            // Query for all creating tags
            const existingTags = await prisma.tag.findMany({
                where: { tag: { in: input.tagCreate.map(c => c.tag) } },
                select: { id: true, tag: true }
            });
            console.log('tag relationshipbuilder existingTags', existingTags);
            // All results should be connects
            const connectTags = existingTags.map(t => ({ id: t.id }));
            // All tags that didn't exist are creates
            const createTags = input.tagCreate.filter(c => !connectTags.some(t => t.id === c.id));
            input.tagConnect = Array.isArray(input.tagConnect) ? [...input.tagConnect, ...connectTags] : createTags;
            input.tagCreate = createTags;
        }
        // Validate create
        if (Array.isArray(input.tagCreate)) {
            for (const tag of input.tagCreate) {
                // Check for valid arguments
                tagCreate.validateSync(input, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(tag.tag, tag.description)) throw new CustomError(CODE.BannedWord);
            }
        }
        // Convert input to Prisma shape
        // Updating/deleting tags is not supported. This must be done in its own query.
        let formattedInput = joinRelationshipToPrisma(input, 'tags', 'tag', isAdd, [], false);
        delete formattedInput.update;
        delete formattedInput.delete;
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async find(
        userId: string | null | undefined, // Of the user making the request, not the tag's creator
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Tag> | null> {
        // Create selector
        const select = selectHelper<Tag, tag>(info, format.toDB);
        // Access database
        let tag = await prisma.tag.findUnique({ where: { id: input.id }, ...select });
        // Return tag with "isStarred" field. This must be queried separately.
        if (!tag) throw new CustomError(CODE.InternalError, 'Tag not found');
        if (!userId) return { ...format.toGraphQL(tag), isStarred: false };
        const star = await prisma.star.findFirst({ where: { byId: userId, tagId: tag.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(tag), isStarred };
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: TagSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // If myId or hidden specified, limit results.
        let idsLimit: string[] | undefined = undefined;
        // Looking for tags the requesting user has starred
        if (userId && input.myTags) {
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
        const search = searcher<TagSortBy, TagSearchInput, Tag, tag>(MODEL_TYPES.Tag, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...where }, { ...input, ids: idsLimit }, info);
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
            console.log('isStarred', isStarred,);
            return { cursor, node: { ...node, isStarred } };
        });
        return searchResults;
    },
    async create(
        userId: string,
        input: TagCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Tag>> {
        // Check for valid arguments
        tagCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.tag, input.description)) throw new CustomError(CODE.BannedWord);
        // Create tag data
        let tagData: { [x: string]: any } = { name: input.tag, description: input.description, createdByUserId: userId };
        // Create tag
        const tag = await prisma.tag.create({
            data: tagData,
            ...selectHelper<Tag, tag>(info, format.toDB)
        })
        // Return tag with "isStarred" field. This will be its default value.
        return { ...format.toGraphQL(tag), isStarred: false };
    },
    async update(
        userId: string,
        input: TagUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Tag>> {
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
            ...selectHelper<Tag, tag>(info, format.toDB)
        });
        // Return tag with "isStarred" field. This must be queried separately.
        const star = await prisma.star.findFirst({ where: { byId: userId, tagId: tag.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(tag), isStarred };
    },
    async deleteMany(userId: string, input: DeleteManyInput): Promise<Count> {
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
        ...tagger(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================