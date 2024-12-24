import { FindByIdInput, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionSearchInput, ScheduleExceptionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsScheduleException = {
    Query: {
        scheduleException: ApiEndpoint<FindByIdInput, FindOneResult<ScheduleException>>;
        scheduleExceptions: ApiEndpoint<ScheduleExceptionSearchInput, FindManyResult<ScheduleException>>;
    },
    Mutation: {
        scheduleExceptionCreate: ApiEndpoint<ScheduleExceptionCreateInput, CreateOneResult<ScheduleException>>;
        scheduleExceptionUpdate: ApiEndpoint<ScheduleExceptionUpdateInput, UpdateOneResult<ScheduleException>>;
    }
}

const objectType = "ScheduleException";
export const ScheduleExceptionEndpoints: EndpointsScheduleException = {
    Query: {
        scheduleException: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        scheduleExceptions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        scheduleExceptionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        scheduleExceptionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
