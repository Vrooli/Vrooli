import { PremiumModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Premium" as const;
export const PremiumFormat: Formatter<PremiumModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};
