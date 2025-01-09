import { FindByIdInput, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionSearchResult, NotificationSubscriptionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsNotificationSubscription = {
    findOne: ApiEndpoint<FindByIdInput, NotificationSubscription>;
    findMany: ApiEndpoint<NotificationSubscriptionSearchInput, NotificationSubscriptionSearchResult>;
    createOne: ApiEndpoint<NotificationSubscriptionCreateInput, NotificationSubscription>;
    updateOne: ApiEndpoint<NotificationSubscriptionUpdateInput, NotificationSubscription>;
}

const objectType = "NotificationSubscription";
export const notificationSubscription: EndpointsNotificationSubscription = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
