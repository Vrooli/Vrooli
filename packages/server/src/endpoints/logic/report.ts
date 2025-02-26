import { FindByIdInput, Report, ReportCreateInput, ReportSearchInput, ReportSearchResult, ReportUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsReport = {
    findOne: ApiEndpoint<FindByIdInput, Report>;
    findMany: ApiEndpoint<ReportSearchInput, ReportSearchResult>;
    createOne: ApiEndpoint<ReportCreateInput, Report>;
    updateOne: ApiEndpoint<ReportUpdateInput, Report>;
}

const objectType = "Report";
export const report: EndpointsReport = {
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
