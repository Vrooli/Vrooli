import { type RunIOSearchInput, type RunIOSearchResult } from "@vrooli/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsRunIO = {
    findMany: ApiEndpoint<RunIOSearchInput, RunIOSearchResult>;
}

const objectType = "RunIO";
export const runIO: EndpointsRunIO = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
