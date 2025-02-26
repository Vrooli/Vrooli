import { BookmarkListSortBy, bookmarkListValidation, MaxObjects } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { BookmarkListFormat } from "../formats.js";
import { BookmarkListModelLogic } from "./types.js";

const __typename = "BookmarkList" as const;
export const BookmarkListModel: BookmarkListModelLogic = ({
    __typename,
    dbTable: "bookmark_list",
    display: () => ({
        label: {
            select: () => ({ id: true, label: true }),
            get: (select) => select.label,
        },
    }),
    format: BookmarkListFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: -1, //TODO
                label: data.label,
                user: { connect: { id: rest.userData.id } },
                bookmarks: await shapeHelper({ relation: "bookmarks", relTypes: ["Connect", "Create"], isOneToOne: false, objectType: "Bookmark", parentRelationshipName: "list", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                label: noNull(data.label),
                bookmarks: await shapeHelper({ relation: "bookmarks", relTypes: ["Connect", "Create", "Update", "Delete"], isOneToOne: false, objectType: "Bookmark", parentRelationshipName: "list", data, ...rest }),
            }),
        },
        yup: bookmarkListValidation,
    },
    search: {
        defaultSort: BookmarkListSortBy.LabelAsc,
        sortBy: BookmarkListSortBy,
        searchFields: {
            bookmarksContainsId: true,
            labelsIds: true,
        },
        searchStringQuery: () => ({
            label: "label",
        }),
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    user: { id: data.userId },
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("BookmarkList", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("BookmarkList", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
