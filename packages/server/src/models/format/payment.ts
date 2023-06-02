import { PaymentModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Payment" as const;
export const PaymentFormat: Formatter<PaymentModelLogic> = {
    gqlRelMap: {
        __typename,
        organization: "Organization",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};
