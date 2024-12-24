import { FindByIdInput, ReportResponse, ReportResponseCreateInput, ReportResponseSearchInput, ReportResponseUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsReportResponse = {
    Query: {
        reportResponse: ApiEndpoint<FindByIdInput, FindOneResult<ReportResponse>>;
        reportResponses: ApiEndpoint<ReportResponseSearchInput, FindManyResult<ReportResponse>>;
    },
    Mutation: {
        reportResponseCreate: ApiEndpoint<ReportResponseCreateInput, CreateOneResult<ReportResponse>>;
        reportResponseUpdate: ApiEndpoint<ReportResponseUpdateInput, UpdateOneResult<ReportResponse>>;
    }
}

const objectType = "ReportResponse";
export const ReportResponseEndpoints: EndpointsReportResponse = {
    Query: {
        reportResponse: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        reportResponses: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        reportResponseCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        reportResponseUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
