import { FindByIdInput, Standard, StandardCreateInput, StandardSearchInput, StandardUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsStandard = {
    Query: {
        standard: ApiEndpoint<FindByIdInput, FindOneResult<Standard>>;
        standards: ApiEndpoint<StandardSearchInput, FindManyResult<Standard>>;
    },
    Mutation: {
        standardCreate: ApiEndpoint<StandardCreateInput, CreateOneResult<Standard>>;
        standardUpdate: ApiEndpoint<StandardUpdateInput, UpdateOneResult<Standard>>;
    }
}

const objectType = "Standard";
export const StandardEndpoints: EndpointsStandard = {
    Query: {
        standard: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        standards: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        standardCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return createOneHelper({ info, input, objectType, req });
        },
        standardUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
