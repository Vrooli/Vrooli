import { FindByIdInput, Payment, PaymentSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
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
        payment: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        payments: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
