import { Count, Tag, TagCreateInput, TagUpdateInput, TagSearchInput, TagSortBy } from "../../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, GraphQLModelType, joinRelationshipToPrisma, modelToGraphQL, PartialGraphQLInfo, RelationshipTypes, removeJoinTablesHelper, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { CustomError } from "../../error";
import { CODE, omit, tagsCreate, tagsUpdate, tagTranslationCreate, tagTranslationUpdate } from "@local/shared";
import { validateProfanity } from "../../utils/censor";
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { genErrorCode } from "../../logger";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { organizations: 'tagged', projects: 'tagged', routines: 'tagged', standards: 'tagged', starredBy: 'user' };
const calculatedFields = ['isStarred', 'isOwn'];
export const tagFormatter = (): FormatConverter<Tag> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Tag,
        'starredBy': GraphQLModelType.User,
    },
    removeCalculatedFields: (partial) => {
        const omitted = omit(partial, calculatedFields);
        // Add createdByUserId field so we can calculate isOwn
        return { ...omitted, createdByUserId: true }
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
        partial: PartialGraphQLInfo,
    ): Promise<RecursivePartial<Tag>[]> {
        console.log('tag addsupplemental fields', JSON.stringify(partial), '\n\n')
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            console.log('getting tag isstarreds', ids, '\n\n')
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, GraphQLModelType.Tag)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
            console.log('got tag isstarreds', JSON.stringify(objects), '\n\n')
        }
        // Query for isOwn
        if (partial.isOwn) objects = objects.map((x) => ({ ...x, isOwn: Boolean(userId) && x.createdByUserId === userId }));
        // Convert Prisma objects to GraphQL objects
        return objects as RecursivePartial<Tag>[];
    },
})

export const tagSearcher = (): Searcher<TagSearchInput> => ({
    defaultSort: TagSortBy.StarsDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [TagSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [TagSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [TagSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [TagSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [TagSortBy.StarsAsc]: { stars: 'asc' },
            [TagSortBy.StarsDesc]: { stars: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        console.log('tags getsearchquery', searchString, languages);
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                { tag: { ...insensitive } },
            ]
        })
    },
    customQueries(input: TagSearchInput): { [x: string]: any } {
        return {
            ...(input.excludeIds !== undefined ? { id: { not: { in: input.excludeIds } } } : {}),
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
        }
    },
})

export const tagVerifier = () => ({
    profanityCheck(data: (TagCreateInput | TagUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.tag));
        TranslationModel().profanityCheck(data);
    },
})

export const tagMutater = (prisma: PrismaType, verifier: ReturnType<typeof tagVerifier>) => ({
    async toDBShape(userId: string | null, data: TagCreateInput | TagUpdateInput): Promise<any> {
        return {
            name: data.tag,
            createdByUserId: userId,
            translations: TranslationModel().relationshipBuilder(userId, data, { create: tagTranslationCreate, update: tagTranslationUpdate }, false),
        }
    },
    /**
     * Maps type of a tag's parent with the unique field
     */
    parentMapper: {
        [GraphQLModelType.Organization]: 'organization_tags_taggedid_tagTag_unique',
        [GraphQLModelType.Project]: 'project_tags_taggedid_tagTag_unique',
        [GraphQLModelType.Routine]: 'routine_tags_taggedid_tagTag_unique',
        [GraphQLModelType.Standard]: 'standard_tags_taggedid_tagTag_unique',
        [GraphQLModelType.TagHidden]: 'user_tags_hidden_userid_tagTag_unique',
    },
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        parentType: keyof typeof this.parentMapper,
        relationshipName: string = 'tags',
    ): Promise<{ [x: string]: any } | undefined> {
        console.log('tag rel builder start', JSON.stringify(input), '\n\n', 'relname', relationshipName);
        // If any tag creates/connects, make sure they exist/not exist
        const initialCreateTags = Array.isArray(input[`${relationshipName}Create`]) ? 
            input[`${relationshipName}Create`].map((c: any) => c.tag) : 
            typeof input[`${relationshipName}Create`] === 'object' ? [input[`${relationshipName}Create`].tag] : 
            [];
        const initialConnectTags = Array.isArray(input[`${relationshipName}Connect`]) ?
            input[`${relationshipName}Connect`] :
            typeof input[`${relationshipName}Connect`] === 'object' ? [input[`${relationshipName}Connect`]] :
            [];
        const bothInitialTags = [...initialCreateTags, ...initialConnectTags];
        console.log('tag rel builder both initial tags', bothInitialTags, '\n\n');
        if (bothInitialTags.length > 0) {
            // Query for all of the tags, to determine which ones exist
            const existingTags = await prisma.tag.findMany({
                where: { tag: { in: bothInitialTags } },
                select: { tag: true }
            });
            console.log('tag rel builder existing tags', JSON.stringify(existingTags), '\n\n');
            // All existing tags are the new connects
            input[`${relationshipName}Connect`] = existingTags.map((t) => ({ tag: t.tag }));
            // All new tags are the new creates
            input[`${relationshipName}Create`] = bothInitialTags.filter((t) => !existingTags.some((et) => et.tag === t)).map((t) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        // Validate create
        if (Array.isArray(input[`${relationshipName}Create`])) {
            const createMany = input[`${relationshipName}Create`];
            tagsCreate.validateSync(createMany, { abortEarly: false });
            verifier.profanityCheck(createMany);
        }
        // Shape disconnects and deletes
        if (Array.isArray(input[`${relationshipName}Disconnect`])) {
            input[`${relationshipName}Disconnect`] = input[`${relationshipName}Disconnect`].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        if (Array.isArray(input[`${relationshipName}Delete`])) {
            input[`${relationshipName}Delete`] = input[`${relationshipName}Delete`].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        // Convert input to Prisma shape
        // Updating/deleting tags is not supported. This must be done in its own query.
        let formattedInput = joinRelationshipToPrisma({
            data: input,
            joinFieldName: 'tag',
            uniqueFieldName: this.parentMapper[parentType],
            childIdFieldName: 'tagTag',
            parentIdFieldName: 'taggedId',
            parentId: input.id ?? null,
            relationshipName,
            isAdd: !Boolean(input.id),
            relExcludes: [RelationshipTypes.update, RelationshipTypes.delete],
            idField: 'tag',
        })
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<TagCreateInput, TagUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId) 
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0112') });
        if (createMany) {
            tagsCreate.validateSync(createMany, { abortEarly: false });
            verifier.profanityCheck(createMany);
            // Check for max tags on object TODO
        }
        if (updateMany) {
            tagsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            verifier.profanityCheck(updateMany.map(u => u.data));
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<TagCreateInput, TagUpdateInput>): Promise<CUDResult<Tag>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShape(input.anonymous ? null : userId, input);
                // Create object
                const currCreated = await prisma.tag.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.tag.findFirst({
                    where: { ...input.where, createdByUserId: userId },
                })
                if (!object) 
                    throw new CustomError(CODE.ErrorUnknown, 'Tag not found', { code: genErrorCode('0113') });
                // Update object
                const currUpdated = await prisma.tag.update({
                    where: input.where,
                    data: await this.toDBShape(input.data.anonymous ? null : userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            const tags = await prisma.tag.findMany({
                where: { id: { in: deleteMany } },
            })
            if (tags.some(t => t.createdByUserId !== userId)) 
                throw new CustomError(CODE.Unauthorized, "You can't delete tags you didn't create", { code: genErrorCode('0114') });
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