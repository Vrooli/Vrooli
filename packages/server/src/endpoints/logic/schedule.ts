import { FindVersionInput, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleUpdateInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsSchedule = {
    Query: {
        schedule: GQLEndpoint<FindVersionInput, FindOneResult<Schedule>>;
        schedules: GQLEndpoint<ScheduleSearchInput, FindManyResult<Schedule>>;
    },
    Mutation: {
        scheduleCreate: GQLEndpoint<ScheduleCreateInput, CreateOneResult<Schedule>>;
        scheduleUpdate: GQLEndpoint<ScheduleUpdateInput, UpdateOneResult<Schedule>>;
    }
}

const objectType = "Schedule";
export const ScheduleEndpoints: EndpointsSchedule = {
    Query: {
        schedule: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        schedules: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        scheduleCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        scheduleUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
