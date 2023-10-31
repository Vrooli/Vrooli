import { FindByIdInput, ReportResponse, ReportResponseCreateInput, ReportResponseSearchInput, ReportResponseUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsReportResponse = {
    Query: {
        reportResponse: GQLEndpoint<FindByIdInput, FindOneResult<ReportResponse>>;
        reportResponses: GQLEndpoint<ReportResponseSearchInput, FindManyResult<ReportResponse>>;
    },
    Mutation: {
        reportResponseCreate: GQLEndpoint<ReportResponseCreateInput, CreateOneResult<ReportResponse>>;
        reportResponseUpdate: GQLEndpoint<ReportResponseUpdateInput, UpdateOneResult<ReportResponse>>;
    }
}

const objectType = "ReportResponse";
export const ReportResponseEndpoints: EndpointsReportResponse = {
    Query: {
        reportResponse: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reportResponses: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        reportResponseCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        reportResponseUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
