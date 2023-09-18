import { PaymentModelLogic } from "../base/types";
import { Formatter } from "../types";

export const PaymentFormat: Formatter<PaymentModelLogic> = {
    gqlRelMap: {
        __typename: "Payment",
        organization: "Organization",
        user: "User",
    },
    prismaRelMap: {
        __typename: "Payment",
        organization: "Organization",
        user: "User",
    },
    countFields: {},
};
