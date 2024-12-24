import { ApiKey, ApiKeyCreateInput, ApiKeyDeleteOneInput, ApiKeyUpdateInput, ApiKeyValidateInput, COOKIE, DeleteOneInput, Success } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { deleteOneHelper } from "../../actions/deletes";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult, UpdateOneResult } from "../../types";

export type EndpointsApiKey = {
    Mutation: {
        apiKeyCreate: ApiEndpoint<ApiKeyCreateInput, CreateOneResult<ApiKey>>;
        apiKeyUpdate: ApiEndpoint<ApiKeyUpdateInput, UpdateOneResult<ApiKey>>;
        apiKeyDeleteOne: ApiEndpoint<ApiKeyDeleteOneInput, Success>;
        apiKeyValidate: ApiEndpoint<ApiKeyValidateInput, ApiKey>;
    }
}

const objectType = "ApiKey";
export const ApiKeyEndpoints: EndpointsApiKey = {
    Mutation: {
        apiKeyCreate: async (_, { input }, { req }, info) => {
            RequestService.assertRequestFrom(req, { isOfficialUser: true });
            await RequestService.get().rateLimit({ maxUser: 10, req });
            return createOneHelper({ info, input, objectType, req });
        },
        apiKeyUpdate: async (_, { input }, { req }, info) => {
            RequestService.assertRequestFrom(req, { isOfficialUser: true });
            await RequestService.get().rateLimit({ maxUser: 10, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        apiKeyDeleteOne: async (_, { input }, { req }) => {
            RequestService.assertRequestFrom(req, { isOfficialUser: true });
            await RequestService.get().rateLimit({ maxUser: 10, req });
            return deleteOneHelper({ input: { id: input.id, objectType } as DeleteOneInput, req });
        },
        apiKeyValidate: async (_, _i, { req, res }) => {
            await RequestService.get().rateLimit({ maxApi: 5000, req });
            // If session is expired
            if (!req.session.apiToken || !req.session.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError("0318", "SessionExpired");
            }
            // TODO
            throw new CustomError("0319", "NotImplemented");
        },
    },
};
