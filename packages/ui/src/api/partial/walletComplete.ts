import { WalletComplete } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "types";

export const walletCompletePartial: GqlPartial<WalletComplete> = {
    __typename: 'WalletComplete',
    full: {
        __define: {
            0: () => relPartial(require('./session').sessionPartial, 'full'),
            1: () => relPartial(require('./wallet').walletPartial, 'common'),
        },
        firstLogIn: true,
        session: { __use: 0 },
        wallet: { __use: 1 }
    }
}