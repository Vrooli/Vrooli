import { WalletComplete } from "@shared/consts";
import { GqlPartial } from "types";

export const walletCompletePartial: GqlPartial<WalletComplete> = {
    __typename: 'WalletComplete',
    full: {
        __define: {
            0: [require('./session').sessionPartial, 'full'],
            1: [require('./wallet').walletPartial, 'common'],
        },
        firstLogIn: true,
        session: { __use: 0 },
        wallet: { __use: 1 }
    }
}