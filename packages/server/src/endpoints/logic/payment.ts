import { FindByIdInput, Payment, PaymentSearchInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";

export type EndpointsPayment = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Payment>>;
    findMany: ApiEndpoint<PaymentSearchInput, FindManyResult<Payment>>;
}

const objectType = "Payment";
export const payment: EndpointsPayment = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
