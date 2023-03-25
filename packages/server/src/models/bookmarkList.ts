import { Prisma } from "@prisma/client";
import { BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListSortBy, BookmarkListUpdateInput, MaxObjects } from "@shared/consts";
import { bookmarkListValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'BookmarkList' as const;
const suppFields = [] as const;
export const BookmarkListModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: BookmarkListCreateInput,
    GqlUpdate: BookmarkListUpdateInput,
    GqlModel: BookmarkList,
    GqlSearch: BookmarkListSearchInput,
    GqlSort: BookmarkListSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.bookmark_listUpsertArgs['create'],
    PrismaUpdate: Prisma.bookmark_listUpsertArgs['update'],
    PrismaModel: Prisma.bookmark_listGetPayload<SelectWrap<Prisma.bookmark_listSelect>>,
    PrismaSelect: Prisma.bookmark_listSelect,
    PrismaWhere: Prisma.bookmark_listWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.bookmark_list,
    display: {
        select: () => ({ id: true, label: true }),
        label: (select) => select.label,
    },
    format: {
        gqlRelMap: {
            __typename,
            bookmarks: 'Bookmark',
        },
        prismaRelMap: {
            __typename,
            bookmarks: 'Bookmark',
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
                ...(await shapeHelper({ relation: 'bookmarks', relTypes: ['Connect', 'Create'], isOneToOne: false, isRequired: false, objectType: 'Bookmark', parentRelationshipName: 'list', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                label: noNull(data.label),
                ...(await shapeHelper({ relation: 'bookmarks', relTypes: ['Connect', 'Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'Bookmark', parentRelationshipName: 'list', data, ...rest })),
            })
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
            label: 'label'
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
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId }
            }),
        },
    },
})