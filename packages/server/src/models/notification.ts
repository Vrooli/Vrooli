import { MaxObjects, Notification, NotificationSearchInput, NotificationSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ModelLogic } from "./types";

const __typename = "Notification" as const;
const suppFields = [] as const;
export const NotificationModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Notification,
    GqlSearch: NotificationSearchInput,
    GqlSort: NotificationSortBy,
    GqlPermission: object,
    PrismaCreate: Prisma.notificationUpsertArgs["create"],
    PrismaUpdate: Prisma.notificationUpsertArgs["update"],
    PrismaModel: Prisma.notificationGetPayload<SelectWrap<Prisma.notificationSelect>>,
    PrismaSelect: Prisma.notificationSelect,
    PrismaWhere: Prisma.notificationWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.notification,
    display: {
        label: {
            select: () => ({ id: true, title: true }),
            get: (select) => select.title,
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
        },
        countFields: {},
    },
    search: {
        defaultSort: NotificationSortBy.DateCreatedDesc,
        sortBy: NotificationSortBy,
        searchFields: {
            createdTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "linkWrapped",
                "titleWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => true,
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
