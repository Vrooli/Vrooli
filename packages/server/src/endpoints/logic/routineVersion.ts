import { FindVersionInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionSearchInput, RoutineVersionSearchResult, RoutineVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsRoutineVersion = {
    findOne: ApiEndpoint<FindVersionInput, RoutineVersion>;
    findMany: ApiEndpoint<RoutineVersionSearchInput, RoutineVersionSearchResult>;
    createOne: ApiEndpoint<RoutineVersionCreateInput, RoutineVersion>;
    updateOne: ApiEndpoint<RoutineVersionUpdateInput, RoutineVersion>;
}

const objectType = "RoutineVersion";
export const routineVersion: EndpointsRoutineVersion = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
