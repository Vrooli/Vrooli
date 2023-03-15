import { MaxObjects } from "@shared/consts";
import { bookmarkListValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'BookmarkList' as const;
const suppFields = [] as const;
export const BookmarkListModel: ModelLogic<any, typeof suppFields> = ({
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