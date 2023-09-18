import { WalletModelLogic } from "../base/types";
import { Formatter } from "../types";

export const WalletFormat: Formatter<WalletModelLogic> = {
    gqlRelMap: {
        __typename: "Wallet",
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename: "Wallet",
        user: "User",
        organization: "Organization",
    },
    countFields: {},
};
