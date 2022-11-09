import { addJoinTablesHelper, addSupplementalFieldsHelper, combineQueries, getSearchStringQueryHelper, joinRelationshipToPrisma, modelToGraphQL, RelationshipTypes, removeJoinTablesHelper, selectHelper } from "./builder";
import { tagsCreate, tagsUpdate, tagTranslationCreate, tagTranslationUpdate } from "@shared/validation";
import { CODE, TagSortBy } from "@shared/consts";
import { omit } from '@shared/utils'; 
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { CustomError, genErrorCode } from "../events";
import { Tag, TagSearchInput, TagCreateInput, TagUpdateInput, Count } from "../schema/types";
import { RecursivePartial, PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { FormatConverter, Searcher, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { organizations: 'tagged', projects: 'tagged', routines: 'tagged', standards: 'tagged', starredBy: 'user' };
const supplementalFields = ['isStarred', 'isOwn'];
export const tagFormatter = (): FormatConverter<Tag, any> => ({
    relationshipMap: {
        '__typename': 'Tag',
        'starredBy': 'User',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    removeSupplementalFields: (partial) => {
        const omitted = omit(partial, supplementalFields);
        // Add createdByUserId field so we can calculate isOwn, and id field for add supplemental groupbyid
        return { ...omitted, createdByUserId: true, id: true }
    },
    async addSupplementalFields({ objects, partial, prisma, userId }): Promise<RecursivePartial<Tag>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Tag')],
                ['isOwn', async () => objects.map((x) => Boolean(userId) && x.createdByUserId === userId)],
            ]
        });
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
        return getSearchStringQueryHelper({ searchString,
            resolver: ({ insensitive }) => ({ 
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { tag: { ...insensitive } },
                ]
            })
        })
    },
    customQueries(input: TagSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.excludeIds !== undefined ? { id: { not: { in: input.excludeIds } } } : {}),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
        ])
    },
})

export const tagVerifier = () => ({
    profanityCheck(data: (TagCreateInput | TagUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => d.tag));
        TranslationModel.profanityCheck(data);
    },
})

export const tagMutater = (prisma: PrismaType) => ({
    toDBShapeBase(userId: string, data: TagCreateInput | TagUpdateInput) {
        return {
            tag: data.tag,
            createdByUserId: userId,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: tagTranslationCreate, update: tagTranslationUpdate }, false),
        }
    },
    async toDBShapeCreate(userId: string, data: TagCreateInput): Promise<Prisma.tagUpsertArgs['create']> {
        return {
            ...this.toDBShapeBase(userId, data),
        }
    },
    async toDBShapeUpdate(userId: string, data: TagUpdateInput): Promise<Prisma.tagUpsertArgs['update']> {
        return {
            ...this.toDBShapeBase(userId, data),
        }
    },
    /**
     * Maps type of a tag's parent with the unique field
     */
    parentMapper: {
        'Organization': 'organization_tags_taggedid_tagTag_unique',
        'Project': 'project_tags_taggedid_tagTag_unique',
        'Routine': 'routine_tags_taggedid_tagTag_unique',
        'Standard': 'standard_tags_taggedid_tagTag_unique',
        'TagHidden': 'user_tags_hidden_userid_tagTag_unique',
    },
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        parentType: keyof typeof this.parentMapper,
        relationshipName: string = 'tags',
    ): Promise<{ [x: string]: any } | undefined> {
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
        if (bothInitialTags.length > 0) {
            // Query for all of the tags, to determine which ones exist
            const existingTags = await prisma.tag.findMany({
                where: { tag: { in: bothInitialTags } },
                select: { tag: true }
            });
            // All existing tags are the new connects
            input[`${relationshipName}Connect`] = existingTags.map((t) => ({ tag: t.tag }));
            // All new tags are the new creates
            input[`${relationshipName}Create`] = bothInitialTags.filter((t) => !existingTags.some((et) => et.tag === t)).map((t) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        // Validate create
        if (Array.isArray(input[`${relationshipName}Create`])) {
            const createMany = input[`${relationshipName}Create`];
            tagsCreate.validateSync(createMany, { abortEarly: false });
            tagVerifier().profanityCheck(createMany);
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
        if (createMany) {
            tagsCreate.validateSync(createMany, { abortEarly: false });
            tagVerifier().profanityCheck(createMany);
            // Check for max tags on object TODO
        }
        if (updateMany) {
            tagsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            tagVerifier().profanityCheck(updateMany.map(u => u.data));
        }
        if (deleteMany) {
            const objects = await prisma.tag.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, createdByUserId: true },
            });
            if (objects.some((o) => o.createdByUserId !== userId)) {
                throw new CustomError(CODE.Unauthorized, "You can't delete tags you didn't create", { code: genErrorCode('0114') });
            }
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
                const data = await this.toDBShapeCreate(input.anonymous ? null : userId, input);
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
                    data: await this.toDBShapeUpdate(input.data.anonymous ? null : userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.tag.deleteMany({
                where: { id: { in: deleteMany } },
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

export const TagModel = ({
    prismaObject: (prisma: PrismaType) => prisma.tag,
    format: tagFormatter(),
    mutate: tagMutater,
    search: tagSearcher(),
    type: 'Tag' as GraphQLModelType,
    verify: tagVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================