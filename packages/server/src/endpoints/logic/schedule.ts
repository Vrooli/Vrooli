import { FindVersionInput, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsSchedule = {
    findOne: ApiEndpoint<FindVersionInput, FindOneResult<Schedule>>;
    findMany: ApiEndpoint<ScheduleSearchInput, FindManyResult<Schedule>>;
    createOne: ApiEndpoint<ScheduleCreateInput, CreateOneResult<Schedule>>;
    updateOne: ApiEndpoint<ScheduleUpdateInput, UpdateOneResult<Schedule>>;
}

const objectType = "Schedule";
export const schedule: EndpointsSchedule = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
