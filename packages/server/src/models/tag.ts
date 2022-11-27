import { tagsCreate, tagsUpdate } from "@shared/validation";
import { TagSortBy } from "@shared/consts";
import { StarModel } from "./star";
import { Tag, TagSearchInput, TagCreateInput, TagUpdateInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { combineQueries, relationshipBuilderHelper } from "../builders";
import { translationRelationshipBuilder } from "../utils";

type SupplementalFields = 'isStarred' | 'isOwn';
const formatter = (): Formatter<Tag, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Tag',
        starredBy: 'User',
    },
    joinMap: { organizations: 'tagged', projects: 'tagged', routines: 'tagged', standards: 'tagged', starredBy: 'user' },
    supplemental: {
        graphqlFields: ['isStarred', 'isOwn'],
        dbFields: ['createdByUserId', 'id'],
        toGraphQL: ({ ids, objects, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'Tag')],
            ['isOwn', async () => objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id)],
        ],
    },
})

const searcher = (): Searcher<
    TagSearchInput,
    TagSortBy,
    Prisma.tagOrderByWithRelationInput,
    Prisma.tagWhereInput
> => ({
    defaultSort: TagSortBy.StarsDesc,
    sortMap: {
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        StarsAsc: { stars: 'asc' },
        StarsDesc: { stars: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
            { tag: { ...insensitive } },
        ]
    }),
    customQueries(input) {
        return combineQueries([
            (input.excludeIds !== undefined ? { id: { not: { in: input.excludeIds } } } : {}),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
        ])
    },
})

const validator = (): Validator<
    TagCreateInput,
    TagUpdateInput,
    Tag,
    Prisma.tagGetPayload<{ select: { [K in keyof Required<Prisma.tagSelect>]: true } }>,
    any,
    Prisma.tagSelect,
    Prisma.tagWhereInput
> => ({
    validateMap: { __typename: 'Tag' },
    isTransferable: false,
    maxObjects: {
        User: 10000,
        Organization: 0,
    },
    permissionsSelect: () => ({ id: true }),
    permissionResolvers: () => [],
    owner: () => ({}),
    isDeleted: () => false,
    isPublic: () => true,
    profanityFields: ['tag'],
    visibility: {
        private: {},
        public: {},
        owner: () => ({}),
    },
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: TagCreateInput | TagUpdateInput, isAdd: boolean) => {
    return {
        tag: data.tag,
        createdByUserId: userData.id,
        translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}

const mutater = (): Mutater<
    Tag,
    { graphql: TagCreateInput, db: Prisma.tagUpsertArgs['create'] },
    { graphql: TagUpdateInput, db: Prisma.tagUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, true),
        update: async ({ data, prisma, userData }) => await shapeBase(prisma, userData, data, false),
    },
    yup: { create: tagsCreate, update: tagsUpdate },
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
    /**
     * Tags are a special case, so they require a custom relationship builder. 
     * This function supports the joint table and connecting when a create already exists, 
     */
    async tagRelationshipBuilder(
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
})

export const TagModel = ({
    delegate: (prisma: PrismaType) => prisma.tag,
    format: formatter(),
    mutate: mutater,
    search: searcher(),
    type: 'Tag' as GraphQLModelType,
    validate: validator(),
})