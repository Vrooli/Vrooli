import { EmailModelLogic } from "../base/types";
import { Formatter } from "../types";

export const EmailFormat: Formatter<EmailModelLogic> = {
    gqlRelMap: {
        __typename: "Email",
    },
    prismaRelMap: {
        __typename: "Email",
        user: "User",
    },
    countFields: {},
};
