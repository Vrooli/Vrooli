import { wallet_update } from "../generated";
import { WalletEndpoints } from "../logic/wallet";
import { setupRoutes } from "./base";

export const WalletRest = setupRoutes({
    "/wallet/:id": {
        put: [WalletEndpoints.Mutation.walletUpdate, wallet_update],
    },
});
