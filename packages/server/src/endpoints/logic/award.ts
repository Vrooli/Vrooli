import { AwardSearchInput, AwardSearchResult, VisibilityType } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsAward = {
    findMany: ApiEndpoint<AwardSearchInput, AwardSearchResult>;
}

const objectType = "Award";
export const award: EndpointsAward = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
};
