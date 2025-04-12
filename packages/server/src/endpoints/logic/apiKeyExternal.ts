import { ApiKeyExternal, ApiKeyExternalCreateInput, ApiKeyExternalUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsApiKeyExternal = {
    createOne: ApiEndpoint<ApiKeyExternalCreateInput, ApiKeyExternal>;
    updateOne: ApiEndpoint<ApiKeyExternalUpdateInput, ApiKeyExternal>;
}

const objectType = "ApiKeyExternal";
export const apiKeyExternal: EndpointsApiKeyExternal = {
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
};
