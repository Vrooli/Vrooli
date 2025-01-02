import { FindVersionInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionSearchInput, RoutineVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsRoutineVersion = {
    findOne: ApiEndpoint<FindVersionInput, FindOneResult<RoutineVersion>>;
    findMany: ApiEndpoint<RoutineVersionSearchInput, FindManyResult<RoutineVersion>>;
    createOne: ApiEndpoint<RoutineVersionCreateInput, CreateOneResult<RoutineVersion>>;
    updateOne: ApiEndpoint<RoutineVersionUpdateInput, UpdateOneResult<RoutineVersion>>;
}

const objectType = "RoutineVersion";
export const routineVersion: EndpointsRoutineVersion = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
