import { type ApiKeyExternal, type ApiKeyExternalCreateInput, type ApiKeyExternalUpdateInput } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

export type EndpointsApiKeyExternal = {
    createOne: ApiEndpoint<ApiKeyExternalCreateInput, ApiKeyExternal>;
    updateOne: ApiEndpoint<ApiKeyExternalUpdateInput, ApiKeyExternal>;
}

export const apiKeyExternal: EndpointsApiKeyExternal = createStandardCrudEndpoints({
    objectType: "ApiKeyExternal",
    endpoints: {
        createOne: {
            rateLimit: { maxUser: 25 },
            permissions: { hasWriteAuthPermissions: true },
            customImplementation: async ({ input, req, info }) => {
                RequestService.assertRequestFrom(req, { isOfficialUser: true });
                const { createOneHelper } = await import("../../actions/creates.js");
                return createOneHelper({ info, input, objectType: "ApiKeyExternal", req });
            },
        },
        updateOne: {
            rateLimit: { maxUser: 25 },
            permissions: { hasWriteAuthPermissions: true },
            customImplementation: async ({ input, req, info }) => {
                RequestService.assertRequestFrom(req, { isOfficialUser: true });
                const { updateOneHelper } = await import("../../actions/updates.js");
                return updateOneHelper({ info, input, objectType: "ApiKeyExternal", req });
            },
        },
    },
});
