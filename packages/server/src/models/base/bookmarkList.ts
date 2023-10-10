import { BookmarkListSortBy, bookmarkListValidation, MaxObjects } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils";
import { BookmarkListFormat } from "../formats";
import { BookmarkListModelLogic } from "./types";

const __typename = "BookmarkList" as const;
export const BookmarkListModel: BookmarkListModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.bookmark_list,
    display: {
        label: {
            select: () => ({ id: true, label: true }),
            get: (select) => select.label,
        },
    },
    format: BookmarkListFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: -1, //TODO
                label: data.label,
                user: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "bookmarks", relTypes: ["Connect", "Create"], isOneToOne: false, isRequired: false, objectType: "Bookmark", parentRelationshipName: "list", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                label: noNull(data.label),
                ...(await shapeHelper({ relation: "bookmarks", relTypes: ["Connect", "Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "Bookmark", parentRelationshipName: "list", data, ...rest })),
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
    validate: {
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
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    },
});
