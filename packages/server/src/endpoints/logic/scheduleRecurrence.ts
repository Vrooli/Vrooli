import { FindByIdInput, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceSearchInput, ScheduleRecurrenceUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsScheduleRecurrence = {
    Query: {
        scheduleRecurrence: GQLEndpoint<FindByIdInput, FindOneResult<ScheduleRecurrence>>;
        scheduleRecurrences: GQLEndpoint<ScheduleRecurrenceSearchInput, FindManyResult<ScheduleRecurrence>>;
    },
    Mutation: {
        scheduleRecurrenceCreate: GQLEndpoint<ScheduleRecurrenceCreateInput, CreateOneResult<ScheduleRecurrence>>;
        scheduleRecurrenceUpdate: GQLEndpoint<ScheduleRecurrenceUpdateInput, UpdateOneResult<ScheduleRecurrence>>;
    }
}

const objectType = "ScheduleRecurrence";
export const ScheduleRecurrenceEndpoints: EndpointsScheduleRecurrence = {
    Query: {
        scheduleRecurrence: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        scheduleRecurrences: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        scheduleRecurrenceCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        scheduleRecurrenceUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
