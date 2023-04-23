import { rel } from "../utils";
export const walletComplete = {
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
//# sourceMappingURL=walletComplete.js.map