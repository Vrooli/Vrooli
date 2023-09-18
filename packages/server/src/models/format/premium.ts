import { PremiumModelLogic } from "../base/types";
import { Formatter } from "../types";

export const PremiumFormat: Formatter<PremiumModelLogic> = {
    gqlRelMap: {
        __typename: "Premium",
    },
    prismaRelMap: {
        __typename: "Premium",
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};
