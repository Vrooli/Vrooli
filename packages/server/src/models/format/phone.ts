import { PhoneModelLogic } from "../base/types";
import { Formatter } from "../types";

export const PhoneFormat: Formatter<PhoneModelLogic> = {
    gqlRelMap: {
        __typename: "Phone",
    },
    prismaRelMap: {
        __typename: "Phone",
        user: "User",
    },
    countFields: {},
};
