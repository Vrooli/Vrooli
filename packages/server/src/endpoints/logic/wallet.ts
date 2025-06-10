import { type Wallet, type WalletUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsWallet = {
    updateOne: ApiEndpoint<WalletUpdateInput, Wallet>;
}

export const wallet: EndpointsWallet = createStandardCrudEndpoints({
    objectType: "Wallet",
    endpoints: {
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: { hasWriteAuthPermissions: true },
        },
    },
});
