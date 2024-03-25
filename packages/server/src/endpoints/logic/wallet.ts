import { Wallet, WalletUpdateInput } from "@local/shared";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsWallet = {
    Mutation: {
        walletUpdate: GQLEndpoint<WalletUpdateInput, UpdateOneResult<Wallet>>;
    }
}

const objectType = "Wallet";
export const WalletEndpoints: EndpointsWallet = {
    Mutation: {
        walletUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
