import { WalletComplete } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const walletComplete: GqlPartial<WalletComplete> = {
    __typename: 'WalletComplete',
    full: {
        __define: {
            0: async () => rel((await import('./session')).session, 'full'),
            1: async () => rel((await import('./wallet')).wallet, 'common'),
        },
        firstLogIn: true,
        session: { __use: 0 },
        wallet: { __use: 1 }
    }
}