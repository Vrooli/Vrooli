import { WalletComplete } from "@shared/consts";
import { GqlPartial } from "types";
import { sessionPartial } from "./session";
import { walletPartial } from "./wallet";

export const walletCompletePartial: GqlPartial<WalletComplete> = {
    __typename: 'WalletComplete',
    full: {
        __define: {
            0: [sessionPartial, 'full'],
            1: [walletPartial, 'full'],
        },
        firstLogIn: true,
        session: { __use: 0 },
        wallet: { __use: 1 }
    }
}