import { Code, CodeCreateInput, CodeSearchInput, CodeUpdateInput, FindByIdInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsCode = {
    Query: {
        code: GQLEndpoint<FindByIdInput, FindOneResult<Code>>;
        codes: GQLEndpoint<CodeSearchInput, FindManyResult<Code>>;
    },
    Mutation: {
        codeCreate: GQLEndpoint<CodeCreateInput, CreateOneResult<Code>>;
        codeUpdate: GQLEndpoint<CodeUpdateInput, UpdateOneResult<Code>>;
    }
}

const objectType = "Code";
export const CodeEndpoints: EndpointsCode = {
    Query: {
        code: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        codes: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        codeCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        codeUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
