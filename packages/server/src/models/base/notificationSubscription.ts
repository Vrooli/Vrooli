import { GqlModelType, MaxObjects, NotificationSubscriptionSortBy, notificationSubscriptionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { subscribableMapper } from "../../events/subscriber";
import { defaultPermissions } from "../../utils";
import { NotificationSubscriptionFormat } from "../formats";
import { NotificationSubscriptionModelLogic } from "./types";

const __typename = "NotificationSubscription" as const;
export const NotificationSubscriptionModel: NotificationSubscriptionModelLogic = ({
    __typename,
    dbTable: "notification_subscription",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(subscribableMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as GqlModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(subscribableMapper)) {
                    if (select[value]) return ModelMap.get(key as GqlModelType).display().label.get(select[value], languages);
                }
                return "";
            },
        },
    }),
    format: NotificationSubscriptionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
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
                ...Object.entries(subscribableMapper).map(([key, value]) => ({ [value]: ModelMap.getLogic(["search"], key as GqlModelType).search.searchStringQuery() })),
            ],
        }),
        /**
         * Extra protection to ensure only you can see your own subscriptions
         */
        customQueryData: (_, user) => ({ subscriber: { id: user!.id } }),
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
            private: {},
            public: {},
            owner: (userId) => ({
                subscriber: { id: userId },
            }),
        },
    }),
});
