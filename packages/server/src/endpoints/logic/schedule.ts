import { FindByPublicIdInput, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleSearchResult, ScheduleUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsSchedule = {
    findOne: ApiEndpoint<FindByPublicIdInput, Schedule>;
    findMany: ApiEndpoint<ScheduleSearchInput, ScheduleSearchResult>;
    createOne: ApiEndpoint<ScheduleCreateInput, Schedule>;
    updateOne: ApiEndpoint<ScheduleUpdateInput, Schedule>;
}

const objectType = "Schedule";
export const schedule: EndpointsSchedule = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
