import { ApiKey, ApiKeyCreateInput, ApiKeyDeleteOneInput, ApiKeyUpdateInput, ApiKeyValidateInput, COOKIE, DeleteOneInput, Success } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { deleteOneHelper } from "../../actions/deletes.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsApiKey = {
    createOne: ApiEndpoint<ApiKeyCreateInput, ApiKey>;
    updateOne: ApiEndpoint<ApiKeyUpdateInput, ApiKey>;
    deleteOne: ApiEndpoint<ApiKeyDeleteOneInput, Success>;
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
    deleteOne: async ({ input }, { req }) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return deleteOneHelper({ input: { id: input.id, objectType } as DeleteOneInput, req });
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
