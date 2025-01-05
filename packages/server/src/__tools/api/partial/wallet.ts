import { Wallet, WalletComplete, WalletInit } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const wallet: GqlPartial<Wallet> = {
    common: {
        id: true,
        name: true,
        publicAddress: true,
        stakingAddress: true,
        verified: true,
    },
};

export const walletInit: GqlPartial<WalletInit> = {
    full: {
        nonce: true,
    },
};

export const walletComplete: GqlPartial<WalletComplete> = {
    full: {
        firstLogIn: true,
        session: async () => rel((await import("./session")).session, "full"),
        wallet: async () => rel((await import("./wallet")).wallet, "common"),
    },
};
