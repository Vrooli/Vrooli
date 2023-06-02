import { EmailModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Email" as const;
export const EmailFormat: Formatter<EmailModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        user: "User",
    },
    countFields: {},
};
