import { WalletModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Wallet" as const;
export const WalletFormat: Formatter<WalletModelLogic> = {
    gqlRelMap: {
        __typename,
        handles: "Handle",
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename,
        handles: "Handle",
        user: "User",
        organization: "Organization",
    },
    countFields: {},
};
