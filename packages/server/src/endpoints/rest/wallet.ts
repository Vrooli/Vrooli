import { wallet_findHandles, wallet_update } from "@local/shared";
import { WalletEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const WalletRest = setupRoutes({
    "/wallet/handles": {
        post: [WalletEndpoints.Query.findHandles, wallet_findHandles],
    },
    "/wallet/:id": {
        put: [WalletEndpoints.Mutation.walletUpdate, wallet_update],
    },
});
