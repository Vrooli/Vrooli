import { BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListSortBy, BookmarkListUpdateInput, bookmarkListValidation, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "BookmarkList" as const;
export const BookmarkListFormat: Formatter<ModelBookmarkListLogic> = {
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
        searchFields: {
            bookmarksContainsId: true,
            labelsIds: true,
        searchStringQuery: () => ({
            label: "label",
        owner: (data) => ({
            User: data.user,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
};
