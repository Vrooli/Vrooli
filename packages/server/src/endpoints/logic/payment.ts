import { FindByIdInput, Payment, PaymentSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../../types";

export type EndpointsPayment = {
    Query: {
        payment: GQLEndpoint<FindByIdInput, FindOneResult<Payment>>;
        payments: GQLEndpoint<PaymentSearchInput, FindManyResult<Payment>>;
    },
}

const objectType = "Payment";
export const PaymentEndpoints: EndpointsPayment = {
    Query: {
        payment: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        payments: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};