import { addJoinTablesHelper, combineQueries, getSearchStringQueryHelper, relationshipBuilderHelper, removeJoinTablesHelper } from "./builder";
import { tagsCreate, tagsUpdate, tagTranslationCreate, tagTranslationUpdate } from "@shared/validation";
import { TagSortBy } from "@shared/consts";
import { StarModel } from "./star";
import { TranslationModel } from "./translation";
import { Tag, TagSearchInput, TagCreateInput, TagUpdateInput } from "../schema/types";
import {  PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

const joinMapper = { organizations: 'tagged', projects: 'tagged', routines: 'tagged', standards: 'tagged', starredBy: 'user' };
type SupplementalFields = 'isStarred' | 'isOwn';
export const tagFormatter = (): FormatConverter<Tag, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Tag',
        starredBy: 'User',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    supplemental: {
        graphqlFields: ['isStarred', 'isOwn'],
        dbFields: ['createdByUserId', 'id'],
        toGraphQL: ({ ids, objects, prisma, userId }) => [
            ['isStarred', async () => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Tag')],
            ['isOwn', async () => objects.map((x) => Boolean(userId) && x.createdByUserId === userId)],
        ],
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

export const tagValidator = (): Validator<
    TagCreateInput,
    TagUpdateInput,
    Tag,
    Prisma.tagGetPayload<{ select: { [K in keyof Required<Prisma.tagSelect>]: true } }>,
    any,
    Prisma.tagSelect,
    Prisma.tagWhereInput
> => ({
    validateMap: {  __typename: 'Tag' },
    permissionsSelect: () => ({ id: true }),
    permissionResolvers: () => [],
    isAdmin: () => false,
    isPublic: () => true,
    profanityFields: ['tag'],
    ownerOrMemberWhere: () => ({}),
})

export const tagMutater = (prisma: PrismaType): Mutater<Tag> => ({
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
        const initialCreate = Array.isArray(data[`${relationshipName}Create`]) ?
            data[`${relationshipName}Create`].map((c: any) => c.tag) :
            typeof data[`${relationshipName}Create`] === 'object' ? [data[`${relationshipName}Create`].tag] :
                [];
        const initialConnect = Array.isArray(data[`${relationshipName}Connect`]) ?
            data[`${relationshipName}Connect`] :
            typeof data[`${relationshipName}Connect`] === 'object' ? [data[`${relationshipName}Connect`]] :
                [];
        const initialCombined = [...initialCreate, ...initialConnect];
        if (initialCombined.length > 0) {
            // Query for all of the tags, to determine which ones exist
            const existing = await prisma.tag.findMany({
                where: { tag: { in: initialCombined } },
                select: { tag: true }
            });
            // All existing tags are the new connects
            data[`${relationshipName}Connect`] = existing.map((t) => ({ tag: t.tag }));
            // All new tags are the new creates
            data[`${relationshipName}Create`] = initialCombined.filter((t) => !existing.some((et) => et.tag === t)).map((t) => typeof t === 'string' ? ({ tag: t }) : t);
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
    validate: tagValidator(),
})