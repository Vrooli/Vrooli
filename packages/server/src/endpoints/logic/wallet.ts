import { Wallet, WalletUpdateInput } from "@local/shared";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, UpdateOneResult } from "../../types";

export type EndpointsWallet = {
    Mutation: {
        walletUpdate: ApiEndpoint<WalletUpdateInput, UpdateOneResult<Wallet>>;
    }
}

const objectType = "Wallet";
export const WalletEndpoints: EndpointsWallet = {
    Mutation: {
        walletUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
