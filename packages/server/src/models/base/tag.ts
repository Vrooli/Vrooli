import { MaxObjects, TagSortBy, getTranslation, tagValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { TagFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, TagModelLogic } from "./types.js";

type TagPre = PreShapeEmbeddableTranslatableResult;

const __typename = "Tag" as const;
export const TagModel: TagModelLogic = ({
    __typename,
    dbTable: "tag",
    dbTranslationTable: "tag_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, tag: true }),
            get: (select) => select.tag,
        },
        embed: {
            select: () => ({ id: true, tag: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, description: true } } }),
            get: ({ tag, translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    description: trans.description,
                    tag,
                }, languages?.[0]);
            },
        },
    }),
    idField: "tag",
    format: TagFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update }): Promise<TagPre> => {
                const maps = preShapeEmbeddableTranslatable<"tag">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            findConnects: async ({ Create }) => {
                const createIds = Create.map(({ node }) => node.id);
                const existingTags = await DbProvider.get().tag.findMany({ where: { tag: { in: createIds } }, select: { tag: true } });
                return createIds.map(id => existingTags.find(x => x.tag === id) ? id : null);
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TagPre;
                return {
                    id: BigInt(data.id),
                    tag: data.tag,
                    createdBy: data.anonymous ? undefined : { connect: { id: rest.userData.id } },
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.tag], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as TagPre;
                return {
                    createdBy: data.anonymous ? { disconnect: true } : undefined,
                    tag: noNull(data.tag),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.tag], data, ...rest }),
                };
            },
        },
        yup: tagValidation,
    },
    search: {
        defaultSort: TagSortBy.EmbedTopDesc,
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
            suppFields: SuppFields[__typename],
            dbFields: ["createdById", "id"],
            getSuppFields: async ({ ids, objects, userData }) => ({
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
            own: null, // Search method disabled, since no one owns site stats
            // Always public, os it's the same as "public"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("StatsSite", "Public", data);
            },
            ownPrivate: null, // Search method disabled, since no one owns site stats
            ownPublic: null, // Search method disabled, since no one owns site stats
            public: function getVisibilityPublic() {
                return {}; // Intentionally empty, since site stats are always public
            },
        },
    }),
});
