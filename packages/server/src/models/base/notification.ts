import { MaxObjects, NotificationSortBy } from "@local/shared";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { NotificationFormat } from "../formats.js";
import { type NotificationModelLogic } from "./types.js";

const __typename = "Notification" as const;
export const NotificationModel: NotificationModelLogic = ({
    __typename,
    dbTable: "notification",
    display: () => ({
        label: {
            select: () => ({ id: true, title: true }),
            get: (select) => select.title,
        },
    }),
    format: NotificationFormat,
    search: {
        defaultSort: NotificationSortBy.DateCreatedDesc,
        sortBy: NotificationSortBy,
        searchFields: {
            createdTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "linkWrapped",
                "titleWrapped",
            ],
        }),
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => true,
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
                return useVisibility("Notification", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Notification", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
