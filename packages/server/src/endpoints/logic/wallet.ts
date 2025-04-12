import { Wallet, WalletUpdateInput } from "@local/shared";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsWallet = {
    updateOne: ApiEndpoint<WalletUpdateInput, Wallet>;
}

const objectType = "Wallet";
export const wallet: EndpointsWallet = {
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
