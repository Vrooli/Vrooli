import { FindByIdInput, Report, ReportCreateInput, ReportSearchInput, ReportUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsReport = {
    Query: {
        report: GQLEndpoint<FindByIdInput, FindOneResult<Report>>;
        reports: GQLEndpoint<ReportSearchInput, FindManyResult<Report>>;
    },
    Mutation: {
        reportCreate: GQLEndpoint<ReportCreateInput, CreateOneResult<Report>>;
        reportUpdate: GQLEndpoint<ReportUpdateInput, UpdateOneResult<Report>>;
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
