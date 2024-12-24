import { Code, CodeCreateInput, CodeSearchInput, CodeUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsCode = {
    Query: {
        code: ApiEndpoint<FindByIdInput, FindOneResult<Code>>;
        codes: ApiEndpoint<CodeSearchInput, FindManyResult<Code>>;
    },
    Mutation: {
        codeCreate: ApiEndpoint<CodeCreateInput, CreateOneResult<Code>>;
        codeUpdate: ApiEndpoint<CodeUpdateInput, UpdateOneResult<Code>>;
    }
}

const objectType = "Code";
export const CodeEndpoints: EndpointsCode = {
    Query: {
        code: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        codes: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        codeCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        codeUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
