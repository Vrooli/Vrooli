import { BookmarkListSortBy, MaxObjects } from "@local/consts";
import { bookmarkListValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
const __typename = "BookmarkList";
const suppFields = [];
export const BookmarkListModel = ({
    __typename,
    delegate: (prisma) => prisma.bookmark_list,
    display: {
        select: () => ({ id: true, label: true }),
        label: (select) => select.label,
    },
    format: {
        gqlRelMap: {
            __typename,
            bookmarks: "Bookmark",
        },
        prismaRelMap: {
            __typename,
            bookmarks: "Bookmark",
        },
        countFields: {
            bookmarksCount: true,
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: -1,
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
            User: data.user,
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
//# sourceMappingURL=bookmarkList.js.map