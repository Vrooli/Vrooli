import { FindByIdInput, Payment, PaymentSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
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
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        payments: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
};
