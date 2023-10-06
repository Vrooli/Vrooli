import { ApiKey, ApiKeyCreateInput, ApiKeyDeleteOneInput, ApiKeyUpdateInput, ApiKeyValidateInput, COOKIE, Success } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { deleteOneHelper } from "../../actions/deletes";
import { updateHelper } from "../../actions/updates";
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
        apiKeyCreate: async (_, { input }, { prisma, req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        apiKeyUpdate: async (_, { input }, { prisma, req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ maxUser: 10, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        apiKeyDeleteOne: async (_, { input }, { prisma, req }) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ maxUser: 10, req });
            return deleteOneHelper({ input, objectType, prisma, req });
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
