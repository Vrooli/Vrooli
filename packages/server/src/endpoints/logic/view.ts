import { type ViewSearchInput, type ViewSearchResult } from "@vrooli/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsView = {
    findMany: ApiEndpoint<ViewSearchInput, ViewSearchResult>;
}

export const view: EndpointsView = createStandardCrudEndpoints({
    objectType: "View",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.VERY_HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
            customImplementation: async ({ input, req, info }) => {
                const userData = RequestService.assertRequestFrom(req, { isUser: true });
                return readManyHelper({ info, input, objectType: "View", req, additionalQueries: { byId: userData.id } });
            },
        },
    },
});
