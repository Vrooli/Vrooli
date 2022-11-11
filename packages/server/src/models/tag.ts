import { addJoinTablesHelper, addSupplementalFieldsHelper, combineQueries, getSearchStringQueryHelper, modelToGraphQL, relationshipBuilderHelper, RelationshipTypes, removeJoinTablesHelper, selectHelper } from "./builder";
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
import { cudHelper } from "./actions";

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
        return getSearchStringQueryHelper({
            searchString,
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
    shapeBase(userId: string | null, data: TagCreateInput | TagUpdateInput) {
        return {
            tag: data.tag,
            createdByUserId: userId,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: tagTranslationCreate, update: tagTranslationUpdate }, false),
        }
    },
    async shapeCreate(userId: string | null, data: TagCreateInput): Promise<Prisma.tagUpsertArgs['create']> {
        return {
            ...this.shapeBase(userId, data),
        }
    },
    async shapeUpdate(userId: string | null, data: TagUpdateInput): Promise<Prisma.tagUpsertArgs['update']> {
        return {
            ...this.shapeBase(userId, data),
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
        data: { [x: string]: any },
        parentType: keyof typeof this.parentMapper,
        relationshipName: string = 'tags',
    ): Promise<{ [x: string]: any } | undefined> {
        // Tags get special logic because they are treated as strings in GraphQL, 
        // instead of a normal relationship object
        // If any tag creates/connects, make sure they exist/not exist
        const initialCreateTags = Array.isArray(data[`${relationshipName}Create`]) ?
            data[`${relationshipName}Create`].map((c: any) => c.tag) :
            typeof data[`${relationshipName}Create`] === 'object' ? [data[`${relationshipName}Create`].tag] :
                [];
        const initialConnectTags = Array.isArray(data[`${relationshipName}Connect`]) ?
            data[`${relationshipName}Connect`] :
            typeof data[`${relationshipName}Connect`] === 'object' ? [data[`${relationshipName}Connect`]] :
                [];
        const bothInitialTags = [...initialCreateTags, ...initialConnectTags];
        if (bothInitialTags.length > 0) {
            // Query for all of the tags, to determine which ones exist
            const existingTags = await prisma.tag.findMany({
                where: { tag: { in: bothInitialTags } },
                select: { tag: true }
            });
            // All existing tags are the new connects
            data[`${relationshipName}Connect`] = existingTags.map((t) => ({ tag: t.tag }));
            // All new tags are the new creates
            data[`${relationshipName}Create`] = bothInitialTags.filter((t) => !existingTags.some((et) => et.tag === t)).map((t) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        // Shape disconnects and deletes
        if (Array.isArray(data[`${relationshipName}Disconnect`])) {
            data[`${relationshipName}Disconnect`] = data[`${relationshipName}Disconnect`].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        if (Array.isArray(data[`${relationshipName}Delete`])) {
            data[`${relationshipName}Delete`] = data[`${relationshipName}Delete`].map((t: any) => typeof t === 'string' ? ({ tag: t }) : t);
        }
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd: false,
            userId,
            idField: 'tag',
            joinData: {
                fieldName: 'tag',
                uniqueFieldName: this.parentMapper[parentType],
                childIdFieldName: 'tagTag',
                parentIdFieldName: 'taggedId',
                parentId: data.id ?? null,
            }
        });
    },
    async cud(params: CUDInput<TagCreateInput, TagUpdateInput>): Promise<CUDResult<Tag>> {
        return cudHelper({
            ...params,
            objectType: 'Tag',
            prisma,
            prismaObject: (p) => p.tag,
            yup: { yupCreate: tagsCreate, yupUpdate: tagsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const TagModel = ({
    prismaObject: (prisma: PrismaType) => prisma.tag,
    format: tagFormatter(),
    mutate: tagMutater,
    search: tagSearcher(),
    type: 'Tag' as GraphQLModelType,
    verify: tagVerifier(),
})