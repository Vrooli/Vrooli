import { FindByIdInput, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsNotificationSubscription = {
    Query: {
        notificationSubscription: GQLEndpoint<FindByIdInput, FindOneResult<NotificationSubscription>>;
        notificationSubscriptions: GQLEndpoint<NotificationSubscriptionSearchInput, FindManyResult<NotificationSubscription>>;
    },
    Mutation: {
        notificationSubscriptionCreate: GQLEndpoint<NotificationSubscriptionCreateInput, CreateOneResult<NotificationSubscription>>;
        notificationSubscriptionUpdate: GQLEndpoint<NotificationSubscriptionUpdateInput, UpdateOneResult<NotificationSubscription>>;
    }
}

const objectType = "NotificationSubscription";
export const NotificationSubscriptionEndpoints: EndpointsNotificationSubscription = {
    Query: {
        notificationSubscription: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        notificationSubscriptions: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        notificationSubscriptionCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        notificationSubscriptionUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
