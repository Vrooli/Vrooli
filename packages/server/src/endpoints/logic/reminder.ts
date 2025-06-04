import { type FindByIdInput, type Reminder, type ReminderCreateInput, type ReminderSearchInput, type ReminderSearchResult, type ReminderUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsReminder = {
    findOne: ApiEndpoint<FindByIdInput, Reminder>;
    findMany: ApiEndpoint<ReminderSearchInput, ReminderSearchResult>;
    createOne: ApiEndpoint<ReminderCreateInput, Reminder>;
    updateOne: ApiEndpoint<ReminderUpdateInput, Reminder>;
}

const objectType = "Reminder";
export const reminder: EndpointsReminder = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPrivatePermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPrivatePermissions: true });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
