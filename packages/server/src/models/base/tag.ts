import { MaxObjects, TagSortBy, tagValidation } from "@local/shared";
import { ModelMap } from ".";
import { prismaInstance } from "../../db/instance";
import { bestTranslation, defaultPermissions } from "../../utils";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString";
import { preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { TagFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, TagModelLogic } from "./types";

const __typename = "Tag" as const;
export const TagModel: TagModelLogic = ({
    __typename,
    delegate: (p) => p.tag,
    display: () => ({
        label: {
            select: () => ({ id: true, tag: true }),
            get: (select) => select.tag,
        },
        embed: {
            select: () => ({ id: true, tag: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, description: true } } }),
            get: ({ tag, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans?.description,
                    tag,
                }, languages[0]);
            },
        },
    }),
    idField: "tag",
    format: TagFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }) => {
                const maps = preShapeEmbeddableTranslatable<"tag">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            findConnects: async ({ Create }) => {
                const createIds = Create.map(({ node }) => node.id);
                const existingTags = await prismaInstance.tag.findMany({ where: { tag: { in: createIds } }, select: { tag: true } });
                return createIds.map(id => existingTags.find(x => x.tag === id) ? id : null);
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                tag: data.tag,
                createdBy: data.anonymous ? undefined : { connect: { id: rest.userData.id } },
                translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.tag], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                createdBy: data.anonymous ? { disconnect: true } : undefined,
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.tag], data, ...rest }),
            }),
        },
        yup: tagValidation,
    },
    search: {
        defaultSort: TagSortBy.Top,
        sortBy: TagSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            excludeIds: true,
            maxBookmarks: true,
            minBookmarks: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => "tagWrapped",
        supplemental: {
            graphqlFields: SuppFields[__typename],
            dbFields: ["createdById", "id"],
            toGraphQL: async ({ ids, objects, userData }) => ({
                you: {
                    isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                    isOwn: objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id),
                },
            }),
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, tag: true }),
        permissionResolvers: defaultPermissions,
        owner: () => ({}),
        isDeleted: () => false,
        isPublic: () => true,
        profanityFields: ["tag"],
        visibility: {
            private: {},
            public: {},
            owner: () => ({}),
        },
    }),
});
