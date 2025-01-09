import { ViewSearchInput, ViewSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsView = {
    findMany: ApiEndpoint<ViewSearchInput, ViewSearchResult>;
}

const objectType = "View";
export const view: EndpointsView = {
    findMany: async ({ input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        return readManyHelper({ info, input, objectType, req, additionalQueries: { byId: userData.id } });
    },
};
