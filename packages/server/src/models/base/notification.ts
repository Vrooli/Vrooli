import { MaxObjects, NotificationSortBy } from "@local/shared";
import { defaultPermissions } from "../../utils";
import { NotificationFormat } from "../format/notification";
import { ModelLogic } from "../types";
import { NotificationModelLogic } from "./types";

const __typename = "Notification" as const;
const suppFields = [] as const;
export const NotificationModel: ModelLogic<NotificationModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.notification,
    display: {
        label: {
            select: () => ({ id: true, title: true }),
            get: (select) => select.title,
        },
    },
    format: NotificationFormat,
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
