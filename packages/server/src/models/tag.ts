import { Count, Tag, TagCreateInput, TagUpdateInput, TagSearchInput, TagSortBy } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, GraphQLModelType, joinRelationshipToPrisma, modelToGraphQL, PartialInfo, RelationshipTypes, removeJoinTablesHelper, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { CustomError } from "../error";
import { CODE, tagCreate, tagUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { StarModel } from "./star";
import _ from "lodash";

export const tagDBFields = ['id', 'created_at', 'updated_at', 'tag', 'description', 'createdByUserId'];

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { organizations: 'tagged', projects: 'tagged', routines: 'tagged', standards: 'tagged', starredBy: 'user' };
export const tagFormatter = (): FormatConverter<Tag> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Tag,
        'starredBy': GraphQLModelType.User,
    },
    removeCalculatedFields: (partial) => {
        let { isStarred, isOwn, ...rest } = partial;
        // Add createdByUserId field so we can calculate isOwn
        return { ...rest, createdByUserId: true }
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialInfo,
    ): Promise<RecursivePartial<Tag>[]> {
        console.log('in tag supplemental fields', partial, objects)
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
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
        console.log('tag supplemental fields COMPLETE', objects)
        return objects as RecursivePartial<Tag>[];
    },
})

export const tagSearcher = (): Searcher<TagSearchInput> => ({
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
    },
})

export const tagVerifier = () => ({
    profanityCheck(data: TagCreateInput | TagUpdateInput): void {
        if (hasProfanity(data.tag, data.description)) throw new CustomError(CODE.BannedWord);
    },
})

export const tagMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShape(userId: string | null, data: TagCreateInput | TagUpdateInput): Promise<any> {
        return {
            name: data.tag,
            description: data.description,
            createdByUserId: userId
        }
    },
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
                verifier.profanityCheck(tag as TagCreateInput);
            }
        }
        // Convert input to Prisma shape
        // Updating/deleting tags is not supported. This must be done in its own query.
        let formattedInput = joinRelationshipToPrisma({ data: input, relationshipName: 'tags', joinFieldName: 'tag', isAdd, relExcludes: [RelationshipTypes.update, RelationshipTypes.delete] })
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<TagCreateInput, TagUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => tagCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check for max tags on object TODO
        }
        if (updateMany) {
            updateMany.forEach(input => tagUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input));
        }
    },
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<TagCreateInput, TagUpdateInput>): Promise<CUDResult<Tag>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShape(input.anonymous ? null : userId, input);
                // Create object
                const currCreated = await prisma.tag.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                const data = await this.toDBShape(input.anonymous ? null : userId, input);
                // Find in database
                let object = await prisma.tag.findFirst({
                    where: { id: input.id, createdByUserId: userId },
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown);
                // Update object
                const currUpdated = await prisma.tag.update({
                    where: { createdByUserId: userId },
                    data,
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            const tags = await prisma.tag.findMany({
                where: { id: { in: deleteMany } },
            })
            if (tags.some(t => t.createdByUserId !== userId)) throw new CustomError(CODE.Unauthorized, "You can't delete tags you didn't create");
            deleted = await prisma.tag.deleteMany({
                where: { id: { in: tags.map(t => t.id) } },
            });
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TagModel(prisma: PrismaType) {
    const prismaObject = prisma.tag;
    const format = tagFormatter();
    const search = tagSearcher();
    const verify = tagVerifier();
    const mutate = tagMutater(prisma, verify);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...verify,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================