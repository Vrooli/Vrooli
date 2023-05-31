import { FindByIdInput, Report, ReportCreateInput, ReportSearchInput, ReportUpdateInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
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
        report: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reports: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        reportCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reportUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
