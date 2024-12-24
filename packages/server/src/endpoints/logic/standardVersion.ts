import { FindVersionInput, StandardVersion, StandardVersionCreateInput, StandardVersionSearchInput, StandardVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsStandardVersion = {
    Query: {
        standardVersion: ApiEndpoint<FindVersionInput, FindOneResult<StandardVersion>>;
        standardVersions: ApiEndpoint<StandardVersionSearchInput, FindManyResult<StandardVersion>>;
    },
    Mutation: {
        standardVersionCreate: ApiEndpoint<StandardVersionCreateInput, CreateOneResult<StandardVersion>>;
        standardVersionUpdate: ApiEndpoint<StandardVersionUpdateInput, UpdateOneResult<StandardVersion>>;
    }
}

const objectType = "StandardVersion";
export const StandardVersionEndpoints: EndpointsStandardVersion = {
    Query: {
        standardVersion: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        standardVersions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        standardVersionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return createOneHelper({ info, input, objectType, req });
        },
        standardVersionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
