import { Wallet, WalletComplete, WalletInit } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const wallet: GqlPartial<Wallet> = {
    __typename: "Wallet",
    common: {
        id: true,
        name: true,
        publicAddress: true,
        stakingAddress: true,
        verified: true,
    },
    full: {},
    list: {},
};

export const walletInit: GqlPartial<WalletInit> = {
    __typename: "WalletInit",
    full: {
        nonce: true,
    },
};

export const walletComplete: GqlPartial<WalletComplete> = {
    __typename: "WalletComplete",
    full: {
        __define: {
            0: async () => rel((await import("./session")).session, "full"),
            1: async () => rel((await import("./wallet")).wallet, "common"),
        },
        firstLogIn: true,
        session: { __use: 0 },
        wallet: { __use: 1 },
    },
};
