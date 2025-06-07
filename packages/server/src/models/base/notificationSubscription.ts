import { MaxObjects, type ModelType, NotificationSubscriptionSortBy, notificationSubscriptionValidation } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { subscribableMapper } from "../../events/subscriber.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { NotificationSubscriptionFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type NotificationSubscriptionModelLogic } from "./types.js";

const __typename = "NotificationSubscription" as const;
export const NotificationSubscriptionModel: NotificationSubscriptionModelLogic = ({
    __typename,
    dbTable: "notification_subscription",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(subscribableMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(subscribableMapper)) {
                    if (select[value]) return ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                }
                return "";
            },
        },
    }),
    format: NotificationSubscriptionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: BigInt(data.id),
                silent: noNull(data.silent),
                subscriber: { connect: { id: rest.userData.id } },
                [subscribableMapper[data.objectType]]: { connect: { id: data.objectConnect } },
            }),
            update: async ({ data }) => ({
                silent: noNull(data.silent),
            }),
        },
        yup: notificationSubscriptionValidation,
    },
    search: {
        defaultSort: NotificationSubscriptionSortBy.DateCreatedDesc,
        sortBy: NotificationSubscriptionSortBy,
        searchFields: {
            createdTimeFrame: true,
            silent: true,
            objectType: true,
            objectId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "titleWrapped",
                ...Object.entries(subscribableMapper).map(([key, value]) => ({ [value]: ModelMap.getLogic(["search"], key as ModelType).search.searchStringQuery() })),
            ],
        }),
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.subscriber,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            subscriber: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    subscriber: { id: data.userId },
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("NotificationSubscription", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("NotificationSubscription", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
