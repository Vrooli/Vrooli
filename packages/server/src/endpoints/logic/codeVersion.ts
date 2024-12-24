import { CodeVersion, CodeVersionCreateInput, CodeVersionSearchInput, CodeVersionUpdateInput, FindVersionInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsCodeVersion = {
    Query: {
        codeVersion: ApiEndpoint<FindVersionInput, FindOneResult<CodeVersion>>;
        codeVersions: ApiEndpoint<CodeVersionSearchInput, FindManyResult<CodeVersion>>;
    },
    Mutation: {
        codeVersionCreate: ApiEndpoint<CodeVersionCreateInput, CreateOneResult<CodeVersion>>;
        codeVersionUpdate: ApiEndpoint<CodeVersionUpdateInput, UpdateOneResult<CodeVersion>>;
    }
}

const objectType = "CodeVersion";
export const CodeVersionEndpoints: EndpointsCodeVersion = {
    Query: {
        codeVersion: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        codeVersions: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        codeVersionCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        codeVersionUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
