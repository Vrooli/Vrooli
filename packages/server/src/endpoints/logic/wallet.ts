import { Wallet, WalletUpdateInput } from "@local/shared";
import { updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsWallet = {
    Mutation: {
        walletUpdate: GQLEndpoint<WalletUpdateInput, UpdateOneResult<Wallet>>;
    }
}

const objectType = "Wallet";
export const WalletEndpoints: EndpointsWallet = {
    Mutation: {
        walletUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
