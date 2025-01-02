import { FindByIdInput, ReportResponse, ReportResponseCreateInput, ReportResponseSearchInput, ReportResponseUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsReportResponse = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<ReportResponse>>;
    findMany: ApiEndpoint<ReportResponseSearchInput, FindManyResult<ReportResponse>>;
    createOne: ApiEndpoint<ReportResponseCreateInput, CreateOneResult<ReportResponse>>;
    updateOne: ApiEndpoint<ReportResponseUpdateInput, UpdateOneResult<ReportResponse>>;
}

const objectType = "ReportResponse";
export const reportResponse: EndpointsReportResponse = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
