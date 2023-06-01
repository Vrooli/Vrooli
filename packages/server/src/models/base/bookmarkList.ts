import { BookmarkListSortBy, bookmarkListValidation, MaxObjects } from "@local/shared";
import { noNull, shapeHelper } from "../builders";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { BookmarkListModelLogic, ModelLogic } from "./types";

const __typename = "BookmarkList" as const;
const suppFields = [] as const;
export const BookmarkListModel: ModelLogic<BookmarkListModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.bookmark_list,
    display: {
        label: {
            select: () => ({ id: true, label: true }),
            get: (select) => select.label,
        },
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
