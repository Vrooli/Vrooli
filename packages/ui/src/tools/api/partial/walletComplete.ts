import { WalletComplete } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const walletCompletePartial: GqlPartial<WalletComplete> = {
    __typename: 'WalletComplete',
    full: {
        __define: {
            0: async () => relPartial((await import('./session')).sessionPartial, 'full'),
            1: async () => relPartial((await import('./wallet')).walletPartial, 'common'),
        },
        firstLogIn: true,
        session: { __use: 0 },
        wallet: { __use: 1 }
    }
}