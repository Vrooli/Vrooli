import { FindByIdInput, Payment, PaymentSearchInput, PaymentSearchResult } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsPayment = {
    findOne: ApiEndpoint<FindByIdInput, Payment>;
    findMany: ApiEndpoint<PaymentSearchInput, PaymentSearchResult>;
}

const objectType = "Payment";
export const payment: EndpointsPayment = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
