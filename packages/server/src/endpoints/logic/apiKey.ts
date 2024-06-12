import { ApiKey, ApiKeyCreateInput, ApiKeyDeleteOneInput, ApiKeyUpdateInput, ApiKeyValidateInput, COOKIE, DeleteOneInput, Success } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { deleteOneHelper } from "../../actions/deletes";
import { updateOneHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsApiKey = {
    Mutation: {
        apiKeyCreate: GQLEndpoint<ApiKeyCreateInput, CreateOneResult<ApiKey>>;
        apiKeyUpdate: GQLEndpoint<ApiKeyUpdateInput, UpdateOneResult<ApiKey>>;
        apiKeyDeleteOne: GQLEndpoint<ApiKeyDeleteOneInput, Success>;
        apiKeyValidate: GQLEndpoint<ApiKeyValidateInput, ApiKey>;
    }
}

const objectType = "ApiKey";
export const ApiKeyEndpoints: EndpointsApiKey = {
    Mutation: {
        apiKeyCreate: async (_, { input }, { req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ maxUser: 10, req });
            return createOneHelper({ info, input, objectType, req });
        },
        apiKeyUpdate: async (_, { input }, { req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ maxUser: 10, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        apiKeyDeleteOne: async (_, { input }, { req }) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ maxUser: 10, req });
            return deleteOneHelper({ input: { id: input.id, objectType } as DeleteOneInput, req });
        },
        apiKeyValidate: async (_, { input }, { req, res }) => {
            await rateLimit({ maxApi: 5000, req });
            // If session is expired
            if (!req.session.apiToken || !req.session.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError("0318", "SessionExpired", req.session.languages);
            }
            // TODO
            throw new CustomError("0319", "NotImplemented", req.session.languages);
        },
    },
};
