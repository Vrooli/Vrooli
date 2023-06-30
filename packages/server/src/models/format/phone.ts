import { PhoneModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Phone" as const;
export const PhoneFormat: Formatter<PhoneModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        user: "User",
    },
    countFields: {},
};
