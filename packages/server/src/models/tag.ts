import { Count, DeleteManyInput, FindByIdInput, Tag, TagCountInput, TagCreateInput, TagUpdateInput, TagSearchInput, TagSortBy } from "../schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { addJoinTables, counter, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, joinRelationshipToPrisma, MODEL_TYPES, PaginatedSearchResult, relationshipFormatter, RelationshipTypes, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, tagCreate, tagUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { tag } from "@prisma/client";
import { StarModel } from "./star";
import _ from "lodash";

export const tagDBFields = ['id', 'created_at', 'updated_at', 'tag', 'description', 'createdByUserId'];

//==============================================================
/* #region Custom Components */
//==============================================================

type TagFormatterType = FormatConverter<Tag, tag>;
/**
 * Component for formatting between graphql and prisma types
 */
export const tagFormatter = (): TagFormatterType => {
    const joinMapper = {
        organizations: 'tagged',
        projects: 'tagged',
        routines: 'tagged',
        standards: 'tagged',
        starredBy: 'user',
    };
    return {
        dbShape: (partial: PartialSelectConvert<Tag>): PartialSelectConvert<tag> => {
            let modified = partial;
            console.log('in tag dbShape aaaaaa', partial)
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['starredBy', FormatterMap.User.dbShapeUser],
            ]);
            console.log('in tag dbShape bbbbbb', partial)
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper)
            // Add creator field, so we can calculate isOwn
            console.log('in tag dbShape cccccc', partial)
            return { ...modified, createdByUserId: true };
        },
        dbPrune: (info: InfoType): PartialSelectConvert<tag> => {
            console.log('in tag dbPrune', info)
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            console.log('in tag dbprune aaaaaaa', info)
            // Remove calculated fields
            let { isStarred, osOwn, ...rest } = modified;
            modified = rest;
            // Convert relationships
            console.log('in tag dbprune bbbbbbb', info)
            modified = relationshipFormatter(modified, [
                ['starredBy', FormatterMap.User.dbPruneUser],
            ]);
            console.log('in tag dbPrune complete', info)
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<tag> => {
            return tagFormatter().dbShape(tagFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<tag>): RecursivePartial<Tag> => {
            console.log('in tagFormatter.toGraphQL start', obj);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            let modified = removeJoinTables(obj, joinMapper);
            // Convert relationships
            modified = relationshipFormatter(obj, [
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
            ]);
            // Remove creator field that was added in shape
            let { createdByUserId, ...rest } = modified;
            modified = rest;
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
            console.log('in tagFormatter.toGraphQL end', modified);
            return modified;
        }
    }
}

/**
 * Component for search filters
 */
export const tagSorter = (): Sortable<TagSortBy> => ({
    defaultSort: TagSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [TagSortBy.AlphabeticalAsc]: { tag: 'asc' },
            [TagSortBy.AlphabeticalDesc]: { tag: 'desc' },
            [TagSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [TagSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [TagSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [TagSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [TagSortBy.StarsAsc]: { stars: 'asc' },
            [TagSortBy.StarsDesc]: { stars: 'desc' },
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
const tagger = (format: TagFormatterType, sort: Sortable<TagSortBy>, prisma: PrismaType) => ({
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
                tagCreate.validateSync(tag, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(tag as TagCreateInput);
            }
        }
        // Convert input to Prisma shape
        // Updating/deleting tags is not supported. This must be done in its own query.
        let formattedInput = joinRelationshipToPrisma({ data: input, relationshipName: 'tags', joinFieldName: 'tag', isAdd, relExcludes: [RelationshipTypes.update, RelationshipTypes.delete] })
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async find(
        userId: string | null | undefined, // Of the user making the request, not the tag's creator
        input: FindByIdInput,
        info: InfoType,
    ): Promise<RecursivePartial<Tag> | null> {
        // Create selector
        const select = selectHelper(info, format.selectToDB);
        // Access database
        let tag = await prisma.tag.findUnique({ where: { id: input.id }, ...select });
        // Return tag with "isStarred" field. This must be queried separately.
        if (!tag) throw new CustomError(CODE.InternalError, 'Tag not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [tag], info);
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: TagSearchInput,
        info: InfoType,
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
        const search = searcher<TagSortBy, TagSearchInput, Tag, tag>(MODEL_TYPES.Tag, format.selectToDB, sort, prisma);
        console.log('calling tag search...')
        let searchResults = await search.search({ ...where }, { ...input, ids: idsLimit }, info);
        // Format and add supplemental/calculated fields to each result node
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, info);
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: TagCreateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Tag>> {
        // Check for valid arguments
        tagCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create tag data
        let tagData: { [x: string]: any } = { name: input.tag, description: input.description, createdByUserId: userId };
        // Create tag
        const tag = await prisma.tag.create({
            data: tagData,
            ...selectHelper(info, format.selectToDB)
        })
        // Return tag with "isStarred" and "isOwn" fields. This will be its default value.
        return { ...format.selectToGraphQL(tag), isStarred: false, isOwn: true };
    },
    async update(
        userId: string,
        input: TagUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Tag>> {
        // Check for valid arguments
        tagUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Create tag data
        let tagData: { [x: string]: any } = { name: input.tag, description: input.description };
        // Update tag
        const tag = await prisma.tag.update({
            where: { createdByUserId: userId },
            data: tagData,
            ...selectHelper(info, format.selectToDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [tag], info);
        return formatted[0];
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
    },
    /**
     * Supplemental fields are isOwn and isStarred
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Tag>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        console.log('in tag supplemental fields', partial, ids)
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'tag')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isOwn
        if (partial.isOwn) objects = objects.map((x) => ({ ...x, isOwn: Boolean(userId) && x.createdByUserId === userId }));
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
    },
    profanityCheck(data: TagCreateInput | TagUpdateInput): void {
        if (hasProfanity(data.tag, data.description)) throw new CustomError(CODE.BannedWord);
    },
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


function AddCreatorField(modified: any): any {
    throw new Error("Function not implemented.");
}
//==============================================================
/* #endregion Model */
//==============================================================