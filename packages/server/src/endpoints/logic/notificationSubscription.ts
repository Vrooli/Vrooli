import { FindByIdInput, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsNotificationSubscription = {
    Query: {
        notificationSubscription: ApiEndpoint<FindByIdInput, FindOneResult<NotificationSubscription>>;
        notificationSubscriptions: ApiEndpoint<NotificationSubscriptionSearchInput, FindManyResult<NotificationSubscription>>;
    },
    Mutation: {
        notificationSubscriptionCreate: ApiEndpoint<NotificationSubscriptionCreateInput, CreateOneResult<NotificationSubscription>>;
        notificationSubscriptionUpdate: ApiEndpoint<NotificationSubscriptionUpdateInput, UpdateOneResult<NotificationSubscription>>;
    }
}

const objectType = "NotificationSubscription";
export const NotificationSubscriptionEndpoints: EndpointsNotificationSubscription = {
    Query: {
        notificationSubscription: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        notificationSubscriptions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        notificationSubscriptionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        notificationSubscriptionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
