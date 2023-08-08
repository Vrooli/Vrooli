import { WalletModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Wallet" as const;
export const WalletFormat: Formatter<WalletModelLogic> = {
    gqlRelMap: {
        __typename,
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename,
        user: "User",
        organization: "Organization",
    },
    countFields: {},
};
