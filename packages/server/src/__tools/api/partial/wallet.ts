import { type Wallet, type WalletComplete, type WalletInit } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const wallet: ApiPartial<Wallet> = {
    common: {
        id: true,
        name: true,
        publicAddress: true,
        stakingAddress: true,
        verifiedAt: true,
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
        session: async () => rel((await import("./session.js")).session, "full"),
        wallet: async () => rel((await import("./wallet.js")).wallet, "common"),
    },
};
