import { Phone } from "@local/shared";
import { GqlPartial } from "../types";

export const phone: GqlPartial<Phone> = {
    __typename: "Phone",
    full: {
        id: true,
        phoneNumber: true,
        verified: true,
    },
};
