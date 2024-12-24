import { FindByIdInput, Payment, PaymentSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";

export type EndpointsPayment = {
    Query: {
        payment: ApiEndpoint<FindByIdInput, FindOneResult<Payment>>;
        payments: ApiEndpoint<PaymentSearchInput, FindManyResult<Payment>>;
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
