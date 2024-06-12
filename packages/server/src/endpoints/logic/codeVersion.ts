import { CodeVersion, CodeVersionCreateInput, CodeVersionSearchInput, CodeVersionUpdateInput, FindVersionInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsCodeVersion = {
    Query: {
        codeVersion: GQLEndpoint<FindVersionInput, FindOneResult<CodeVersion>>;
        codeVersions: GQLEndpoint<CodeVersionSearchInput, FindManyResult<CodeVersion>>;
    },
    Mutation: {
        codeVersionCreate: GQLEndpoint<CodeVersionCreateInput, CreateOneResult<CodeVersion>>;
        codeVersionUpdate: GQLEndpoint<CodeVersionUpdateInput, UpdateOneResult<CodeVersion>>;
    }
}

const objectType = "CodeVersion";
export const CodeVersionEndpoints: EndpointsCodeVersion = {
    Query: {
        codeVersion: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        codeVersions: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        codeVersionCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        codeVersionUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
