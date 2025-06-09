import { type FindByIdInput, type Payment, type PaymentSearchInput, type PaymentSearchResult } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsPayment = {
    findOne: ApiEndpoint<FindByIdInput, Payment>;
    findMany: ApiEndpoint<PaymentSearchInput, PaymentSearchResult>;
}

export const payment: EndpointsPayment = createStandardCrudEndpoints({
    objectType: "Payment",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
    },
});
