import { FindVersionInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionSearchInput, RoutineVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsRoutineVersion = {
    Query: {
        routineVersion: ApiEndpoint<FindVersionInput, FindOneResult<RoutineVersion>>;
        routineVersions: ApiEndpoint<RoutineVersionSearchInput, FindManyResult<RoutineVersion>>;
    },
    Mutation: {
        routineVersionCreate: ApiEndpoint<RoutineVersionCreateInput, CreateOneResult<RoutineVersion>>;
        routineVersionUpdate: ApiEndpoint<RoutineVersionUpdateInput, UpdateOneResult<RoutineVersion>>;
    }
}

const objectType = "RoutineVersion";
export const RoutineVersionEndpoints: EndpointsRoutineVersion = {
    Query: {
        routineVersion: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        routineVersions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        routineVersionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        routineVersionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
