import { FindByIdInput, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateHelper } from "../../actions/updates";
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
        notificationSubscription: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        notificationSubscriptions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        notificationSubscriptionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        notificationSubscriptionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
