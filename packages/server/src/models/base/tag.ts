import { MaxObjects, TagSortBy, tagValidation } from "@local/shared";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../utils";
import { getEmbeddableString } from "../utils/embeddings/getEmbeddableString";
import { preShapeEmbeddableTranslatable } from "../utils/preShapeEmbeddableTranslatable";
import { BookmarkModel } from "./bookmark";
import { ModelLogic } from "./types";

const __typename = "Tag" as const;
const suppFields = ["you"] as const;
export const TagModel: ModelLogic<StatsTagModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.tag,
    display: {
        label: {
            select: () => ({ id: true, tag: true }),
            get: (select) => select.tag,
        },
        embed: {
            select: () => ({ id: true, tag: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, description: true } } }),
            get: ({ tag, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans.description,
                    tag,
                }, languages[0]);
            },
        },
    },
    idField: "tag",
    format: {
        gqlRelMap: {
            __typename,
            apis: "Api",
            notes: "Note",
            organizations: "Organization",
            posts: "Post",
            projects: "Project",
            reports: "Report",
            routines: "Routine",
            smartContracts: "SmartContract",
            standards: "Standard",
            bookmarkedBy: "User",
        },
        prismaRelMap: {
            __typename,
            createdBy: "User",
            apis: "Api",
            notes: "Note",
            organizations: "Organization",
            posts: "Post",
            projects: "Project",
            reports: "Report",
            routines: "Routine",
            smartContracts: "SmartContract",
            standards: "Standard",
            bookmarkedBy: "User",
            focusModeFilters: "FocusModeFilter",
        },
        joinMap: {
            apis: "tagged",
            notes: "tagged",
            organizations: "tagged",
            posts: "tagged",
            projects: "tagged",
            reports: "tagged",
            routines: "tagged",
            smartContracts: "tagged",
            standards: "tagged",
            bookmarkedBy: "user",
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ["createdById", "id"],
            toGraphQL: async ({ ids, objects, prisma, userData }) => ({
                you: {
                    isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                    isOwn: objects.map((x) => Boolean(userData) && x.createdByUserId === userData?.id),
                },
            }),
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList }) => {
                const maps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                tag: data.tag,
                createdBy: data.anonymous ? undefined : { connect: { id: rest.userData.id } },
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.tag], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(data.anonymous ? { createdBy: { disconnect: true } } : {}),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.tag], data, ...rest })),
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
        searchStringQuery: () => ({
            OR: [
                "tagWrapped",
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true }),
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
    },
});
