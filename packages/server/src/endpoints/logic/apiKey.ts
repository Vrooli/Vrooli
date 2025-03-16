import { ApiKey, ApiKeyCreateInput, ApiKeyCreated, ApiKeyUpdateInput, ApiKeyValidateInput, COOKIE } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsApiKey = {
    createOne: ApiEndpoint<ApiKeyCreateInput, ApiKeyCreated>;
    updateOne: ApiEndpoint<ApiKeyUpdateInput, ApiKey>;
    validate: ApiEndpoint<ApiKeyValidateInput, ApiKey>;
}

const objectType = "ApiKey";
export const apiKey: EndpointsApiKey = {
    createOne: async ({ input }, { req }, info) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    validate: async (_i, { req, res }) => {
        await RequestService.get().rateLimit({ maxApi: 5000, req });
        // If session is expired
        if (!req.session.apiToken || !req.session.validToken) {
            res.clearCookie(COOKIE.Jwt);
            throw new CustomError("0318", "SessionExpired");
        }
        // TODO
        throw new CustomError("0319", "NotImplemented");
    },
};
