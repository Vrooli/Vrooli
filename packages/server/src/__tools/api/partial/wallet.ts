import { Wallet, WalletComplete, WalletInit } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const wallet: ApiPartial<Wallet> = {
    common: {
        id: true,
        name: true,
        publicAddress: true,
        stakingAddress: true,
        verified: true,
    },
};

export const walletInit: ApiPartial<WalletInit> = {
    full: {
        nonce: true,
    },
};

export const walletComplete: ApiPartial<WalletComplete> = {
    full: {
        firstLogIn: true,
        session: async () => rel((await import("./session")).session, "full"),
        wallet: async () => rel((await import("./wallet")).wallet, "common"),
    },
};
