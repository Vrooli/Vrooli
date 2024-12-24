import { FindByIdInput, Report, ReportCreateInput, ReportSearchInput, ReportUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsReport = {
    Query: {
        report: ApiEndpoint<FindByIdInput, FindOneResult<Report>>;
        reports: ApiEndpoint<ReportSearchInput, FindManyResult<Report>>;
    },
    Mutation: {
        reportCreate: ApiEndpoint<ReportCreateInput, CreateOneResult<Report>>;
        reportUpdate: ApiEndpoint<ReportUpdateInput, UpdateOneResult<Report>>;
    }
}

const objectType = "Report";
export const ReportEndpoints: EndpointsReport = {
    Query: {
        report: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        reports: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        reportCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        reportUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
