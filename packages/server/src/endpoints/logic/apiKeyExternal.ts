import { type ApiKeyExternal, type ApiKeyExternalCreateInput, type ApiKeyExternalUpdateInput } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsApiKeyExternal = {
    createOne: ApiEndpoint<ApiKeyExternalCreateInput, ApiKeyExternal>;
    updateOne: ApiEndpoint<ApiKeyExternalUpdateInput, ApiKeyExternal>;
}

const objectType = "ApiKeyExternal";
export const apiKeyExternal: EndpointsApiKeyExternal = {
    createOne: async ({ input }, { req }, info) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
